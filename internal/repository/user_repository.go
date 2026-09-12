package repository

import (
	"context"
	"fmt"
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
