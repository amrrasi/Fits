package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/models"
)

// UserRepository handles all persistence for users and sessions.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Users
// ─────────────────────────────────────────────────────────────────────────────

// GetByEmail returns the user with the given email, or ErrNotFound.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`

	u := &models.User{}
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Role, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get user by email: %w", err)
	}
	return u, nil
}

// GetByID returns the user with the given id.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	u := &models.User{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Role, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get user by id: %w", err)
	}
	return u, nil
}

// CreateUser inserts a new user and returns their ID.
func (r *UserRepository) CreateUser(ctx context.Context, email, passwordHash, fullName string, role models.Role) (int64, error) {
	const q = `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, q, email, passwordHash, fullName, string(role)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("repository: create user: %w", err)
	}
	return id, nil
}

// UpdateLastLogin stamps the last login time for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("repository: update last login: %w", err)
	}
	return nil
}

// UpdatePassword updates a user's password hash.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID int64, newHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, newHash,
	)
	if err != nil {
		return fmt.Errorf("repository: update password: %w", err)
	}
	return nil
}

// UpdateRole changes a user's role.
func (r *UserRepository) UpdateRole(ctx context.Context, userID int64, role models.Role) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1`,
		userID, string(role),
	)
	if err != nil {
		return fmt.Errorf("repository: update role: %w", err)
	}
	return nil
}

// DeactivateUser sets is_active=false (soft delete).
func (r *UserRepository) DeactivateUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("repository: deactivate user: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sessions (refresh tokens)
// ─────────────────────────────────────────────────────────────────────────────

// CreateSession stores a new refresh token session.
// tokenHash must be the SHA-256 hash of the raw refresh token.
func (r *UserRepository) CreateSession(ctx context.Context, userID int64, tokenHash, userAgent, ip string, expiresAt time.Time) (string, error) {
	const q = `
		INSERT INTO sessions (user_id, refresh_token, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, q, userID, tokenHash, userAgent, ip, expiresAt).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("repository: create session: %w", err)
	}
	return id, nil
}

// GetSessionByTokenHash returns the session matching the hashed token.
func (r *UserRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	const q = `
		SELECT id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at
		FROM sessions WHERE refresh_token = $1`

	s := &models.Session{}
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&s.ID, &s.UserID, &s.RefreshToken,
		&s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get session: %w", err)
	}
	return s, nil
}

// DeleteSession removes a session by its token hash (logout).
func (r *UserRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("repository: delete session: %w", err)
	}
	return nil
}

// DeleteAllUserSessions removes all sessions for a user (logout everywhere).
func (r *UserRepository) DeleteAllUserSessions(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("repository: delete all user sessions: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes sessions past their expiry (call periodically).
func (r *UserRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("repository: delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sentinel errors
// ─────────────────────────────────────────────────────────────────────────────

// ErrNotFound is returned when a queried row does not exist.
var ErrNotFound = fmt.Errorf("repository: not found")

// ─────────────────────────────────────────────────────────────────────────────
// Audit Logs
// ─────────────────────────────────────────────────────────────────────────────

// AuditLog is a single row from the audit_logs table.
type AuditLog struct {
	ID         int64       `json:"id"`
	UserID     *int64      `json:"user_id"`
	UserEmail  *string     `json:"user_email"`
	Action     string      `json:"action"`
	EntityType string      `json:"entity_type"`
	EntityID   *string     `json:"entity_id"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	IPAddress  *string     `json:"ip_address"`
	RequestID  *string     `json:"request_id"`
	CreatedAt  string      `json:"created_at"`
}

// ListAuditFilter controls filtering for audit logs.
type ListAuditFilter struct {
	UserID     *int64
	Action     string
	EntityType string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

// ListAuditLogs returns paginated audit log entries with optional filters.
func (r *UserRepository) ListAuditLogs(ctx context.Context, f ListAuditFilter) ([]AuditLog, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("a.user_id = $%d", idx))
		args = append(args, *f.UserID)
		idx++
	}
	if f.Action != "" {
		where = append(where, fmt.Sprintf("a.action = $%d", idx))
		args = append(args, f.Action)
		idx++
	}
	if f.EntityType != "" {
		where = append(where, fmt.Sprintf("a.entity_type = $%d", idx))
		args = append(args, f.EntityType)
		idx++
	}
	if f.DateFrom != "" {
		where = append(where, fmt.Sprintf("a.created_at >= $%d", idx))
		args = append(args, f.DateFrom)
		idx++
	}
	if f.DateTo != "" {
		where = append(where, fmt.Sprintf("a.created_at <= $%d", idx))
		args = append(args, f.DateTo)
		idx++
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM audit_logs a WHERE %s", clause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository: count audit logs: %w", err)
	}

	if f.Page < 1 { f.Page = 1 }
	if f.PageSize < 1 { f.PageSize = 50 }
	offset := (f.Page - 1) * f.PageSize

	listArgs := append(args, f.PageSize, offset)
	q := fmt.Sprintf(`
		SELECT a.id, a.user_id, u.email, a.action, a.entity_type, a.entity_id,
		       a.old_value, a.new_value, a.ip_address, a.request_id,
		       TO_CHAR(a.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE %s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d`, clause, idx, idx+1)

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository: list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.UserEmail,
			&l.Action, &l.EntityType, &l.EntityID,
			&l.OldValue, &l.NewValue,
			&l.IPAddress, &l.RequestID, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("repository: scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

// InsertAuditLog writes an audit entry. userID nil = system action.
func (r *UserRepository) InsertAuditLog(ctx context.Context, userID *int64, action, entityType, entityID string, oldVal, newVal interface{}, ip, requestID string) error {
	const q = `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, old_value, new_value, ip_address, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, q, userID, action, entityType, entityID, oldVal, newVal, ip, requestID)
	if err != nil {
		return fmt.Errorf("repository: insert audit log: %w", err)
	}
	return nil
}
