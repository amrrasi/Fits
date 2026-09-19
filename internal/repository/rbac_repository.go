package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/models"
)

// RBACRepository handles roles, permissions, role_permissions, and user_roles.
type RBACRepository struct {
	pool *pgxpool.Pool
}

func NewRBACRepository(pool *pgxpool.Pool) *RBACRepository {
	return &RBACRepository{pool: pool}
}

// GetUserPermissionCodes returns the distinct set of permission codes granted
// to a user through ALL of their assigned roles. This is the single source of
// truth used when issuing a JWT and when gating routes.
func (r *RBACRepository) GetUserPermissionCodes(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.code
		FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id = ur.role_id
		JOIN permissions p       ON p.id = rp.permission_id
		WHERE ur.user_id = $1
		ORDER BY p.code`, userID)
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: get user permissions: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("rbac_repo: scan permission: %w", err)
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// GetUserRoleNames returns the names of every role assigned to a user.
func (r *RBACRepository) GetUserRoleNames(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ro.name
		FROM user_roles ur
		JOIN roles ro ON ro.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY ro.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: get user roles: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("rbac_repo: scan role: %w", err)
		}
		names = append(names, n)
	}
	return names, nil
}

// SetPrimaryBuiltinRole ensures a user holds exactly one of the three built-in
// roles (admin|editor|viewer) — the one named — without touching any other,
// custom role the user may separately have been granted. This is what keeps
// users.role (set via the normal create/update user flow) and the real RBAC
// tables in sync.
func (r *RBACRepository) SetPrimaryBuiltinRole(ctx context.Context, userID int64, roleName string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rbac_repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Remove whichever built-in role(s) the user currently has.
	if _, err := tx.Exec(ctx, `
		DELETE FROM user_roles
		WHERE user_id = $1
		  AND role_id IN (SELECT id FROM roles WHERE name IN ('admin','editor','viewer'))`,
		userID,
	); err != nil {
		return fmt.Errorf("rbac_repo: clear builtin roles: %w", err)
	}

	// Assign the new one.
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT DO NOTHING`,
		userID, roleName,
	); err != nil {
		return fmt.Errorf("rbac_repo: assign builtin role: %w", err)
	}

	return tx.Commit(ctx)
}

// ListRoles returns every role together with its permission codes.
func (r *RBACRepository) ListRoles(ctx context.Context) ([]models.RoleWithPermissions, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, is_builtin, created_at
		FROM roles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: list roles: %w", err)
	}
	defer rows.Close()

	var roles []models.RoleWithPermissions
	for rows.Next() {
		var rd models.RoleDef
		if err := rows.Scan(&rd.ID, &rd.Name, &rd.Description, &rd.IsBuiltin, &rd.CreatedAt); err != nil {
			return nil, fmt.Errorf("rbac_repo: scan role: %w", err)
		}
		roles = append(roles, models.RoleWithPermissions{RoleDef: rd})
	}

	for i := range roles {
		perms, err := r.getRolePermissionCodes(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}
	return roles, nil
}

func (r *RBACRepository) getRolePermissionCodes(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.code`, roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: role permissions: %w", err)
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("rbac_repo: scan role permission: %w", err)
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// ListPermissions returns the full permission catalog.
func (r *RBACRepository) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, description, created_at FROM permissions ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: list permissions: %w", err)
	}
	defer rows.Close()

	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("rbac_repo: scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// GetRoleByName looks up a role's row by name.
func (r *RBACRepository) GetRoleByName(ctx context.Context, name string) (*models.RoleDef, error) {
	const q = `SELECT id, name, description, is_builtin, created_at FROM roles WHERE name=$1`
	rd := &models.RoleDef{}
	err := r.pool.QueryRow(ctx, q, name).Scan(&rd.ID, &rd.Name, &rd.Description, &rd.IsBuiltin, &rd.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rbac_repo: get role by name: %w", err)
	}
	return rd, nil
}

// CreateRole creates a new custom (non-builtin) role.
func (r *RBACRepository) CreateRole(ctx context.Context, name, description string) (int64, error) {
	const q = `INSERT INTO roles (name, description, is_builtin) VALUES ($1, $2, FALSE) RETURNING id`
	var id int64
	if err := r.pool.QueryRow(ctx, q, name, description).Scan(&id); err != nil {
		return 0, fmt.Errorf("rbac_repo: create role: %w", err)
	}
	return id, nil
}

// DeleteRole removes a custom role. Built-in roles cannot be deleted.
func (r *RBACRepository) DeleteRole(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id=$1 AND is_builtin = FALSE`, id)
	if err != nil {
		return fmt.Errorf("rbac_repo: delete role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetRolePermissions replaces a role's entire permission set.
func (r *RBACRepository) SetRolePermissions(ctx context.Context, roleID int64, permissionCodes []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rbac_repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=$1`, roleID); err != nil {
		return fmt.Errorf("rbac_repo: clear role permissions: %w", err)
	}

	for _, code := range permissionCodes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE code=$2
			ON CONFLICT DO NOTHING`, roleID, code,
		); err != nil {
			return fmt.Errorf("rbac_repo: set role permission %q: %w", code, err)
		}
	}

	return tx.Commit(ctx)
}

// AssignUserRole grants an additional role to a user (on top of whatever they
// already have — including their primary built-in role).
func (r *RBACRepository) AssignUserRole(ctx context.Context, userID, roleID int64, assignedBy *int64) error {
	const q = `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`
	if _, err := r.pool.Exec(ctx, q, userID, roleID, assignedBy); err != nil {
		return fmt.Errorf("rbac_repo: assign user role: %w", err)
	}
	return nil
}

// RemoveUserRole revokes a role from a user.
func (r *RBACRepository) RemoveUserRole(ctx context.Context, userID, roleID int64) error {
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID,
	); err != nil {
		return fmt.Errorf("rbac_repo: remove user role: %w", err)
	}
	return nil
}

// GetRoleByID returns a role definition by id (ErrNotFound if missing).
func (r *RBACRepository) GetRoleByID(ctx context.Context, id int64) (*models.RoleDef, error) {
	var rd models.RoleDef
	err := r.pool.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), is_builtin, created_at FROM roles WHERE id = $1`, id).
		Scan(&rd.ID, &rd.Name, &rd.Description, &rd.IsBuiltin, &rd.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return &rd, err
}

// UnknownPermissions returns the codes that do not exist.
func (r *RBACRepository) UnknownPermissions(ctx context.Context, codes []string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT code FROM permissions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		known[c] = true
	}
	var bad []string
	for _, c := range codes {
		if !known[c] {
			bad = append(bad, c)
		}
	}
	return bad, rows.Err()
}
