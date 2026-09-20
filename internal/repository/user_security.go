package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *UserRepository) GetMustChange(ctx context.Context, id int64) (bool, error) {
	var v bool
	err := r.pool.QueryRow(ctx, `SELECT must_change_password FROM users WHERE id = $1`, id).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return v, err
}

func (r *UserRepository) SetMustChange(ctx context.Context, id int64, v bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET must_change_password = $2 WHERE id = $1`, id, v)
	return err
}

// SessionRow is a session as shown to its owner (the token hash is never exposed).
type SessionRow struct {
	ID        string
	TokenHash string
	UserAgent string
	IPAddress string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (r *UserRepository) ListSessions(ctx context.Context, userID int64) ([]SessionRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, refresh_token, user_agent, ip_address, created_at, expires_at
		FROM sessions WHERE user_id = $1 AND expires_at > NOW() ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("user_repo: list sessions: %w", err)
	}
	defer rows.Close()
	var out []SessionRow
	for rows.Next() {
		var s SessionRow
		if err := rows.Scan(&s.ID, &s.TokenHash, &s.UserAgent, &s.IPAddress, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteSessionByID revokes one session, but only if it belongs to userID.
func (r *UserRepository) DeleteSessionByID(ctx context.Context, userID int64, id string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id::text = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteOldAuditLogs applies the retention policy.
func (r *UserRepository) DeleteOldAuditLogs(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM audit_logs WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteUserGuardedForTest removes a row without the last-admin guard (test cleanup only).
func (r *UserRepository) DeleteUserGuardedForTest(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
