package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/amrrasi/fits/internal/models"
)

// ListUsersFilter controls filtering and pagination for ListUsers.
type ListUsersFilter struct {
	Search   string      // partial match on email or full_name
	Role     models.Role // filter by role (empty = all)
	IsActive *bool       // nil = all, true = active only, false = inactive only
	Page     int
	PageSize int
}

// ListUsersResult holds a page of users plus the total count.
type ListUsersResult struct {
	Users []models.User
	Total int
}

// ListUsers returns a filtered, paginated list of users.
func (r *UserRepository) ListUsers(ctx context.Context, f ListUsersFilter) (*ListUsersResult, error) {
	// Build WHERE clauses dynamically
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf("(email ILIKE $%d OR full_name ILIKE $%d)", idx, idx+1))
		pattern := "%" + f.Search + "%"
		args = append(args, pattern, pattern)
		idx += 2
	}
	if f.Role != "" {
		where = append(where, fmt.Sprintf("role = $%d", idx))
		args = append(args, string(f.Role))
		idx++
	}
	if f.IsActive != nil {
		where = append(where, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *f.IsActive)
		idx++
	}

	clause := strings.Join(where, " AND ")

	// Count
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", clause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("repository: count users: %w", err)
	}

	// Rows
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PageSize

	listArgs := append(args, f.PageSize, offset)
	listQ := fmt.Sprintf(`
		SELECT id, email, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, clause, idx, idx+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("repository: list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
			&u.Role, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list users rows: %w", err)
	}

	return &ListUsersResult{Users: users, Total: total}, nil
}

// UpdateUser updates editable fields on a user (full_name, role, is_active).
func (r *UserRepository) UpdateUser(ctx context.Context, id int64, fullName string, role models.Role, isActive bool) error {
	const q = `
		UPDATE users
		SET full_name = $2, role = $3, is_active = $4, updated_at = NOW()
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, fullName, string(role), isActive)
	if err != nil {
		return fmt.Errorf("repository: update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUser hard-deletes a user (cascades to sessions).
// Prefer DeactivateUser for soft deletes.
func (r *UserRepository) DeleteUser(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("repository: delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EmailExists returns true if an account with that email already exists.
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check email: %w", err)
	}
	return exists, nil
}
