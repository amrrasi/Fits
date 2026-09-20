package models

import "time"

type Permission struct {
	ID          int64     `db:"id" json:"id"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type RoleDef struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsBuiltin   bool      `db:"is_builtin" json:"is_builtin"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type RoleWithPermissions struct {
	RoleDef
	Permissions []string `json:"permissions"`
}
