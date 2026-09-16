package models

import "time"

// Permission is a single fine-grained capability, e.g. "files.delete".
// The catalog is seeded by migrations and is not meant to be user-editable
// at runtime — only which permissions belong to which role is editable.
type Permission struct {
	ID          int64     `db:"id" json:"id"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// RoleDef is a row in the roles table (distinct from the legacy `Role` string
// type in auth.go, which remains the label stored on users.role).
type RoleDef struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsBuiltin   bool      `db:"is_builtin" json:"is_builtin"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// RoleWithPermissions is a role together with the permission codes granted to it.
type RoleWithPermissions struct {
	RoleDef
	Permissions []string `json:"permissions"`
}
