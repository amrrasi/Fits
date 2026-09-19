package models

import (
	"time"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (r Role) CanEdit() bool {
	return r == RoleAdmin || r == RoleEditor
}

func (r Role) CanAdmin() bool {
	return r == RoleAdmin
}

type User struct {
	ID           int64      `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	FullName     string     `db:"full_name"`
	Role         Role       `db:"role"`
	IsActive     bool       `db:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type SafeUser struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Role        Role       `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Permissions []string   `json:"permissions,omitempty"`
}

func (u *User) ToSafe() SafeUser {
	return SafeUser{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		Role:        u.Role,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

type Session struct {
	ID           string    `db:"id"`
	UserID       int64     `db:"user_id"`
	RefreshToken string    `db:"refresh_token"` // stored as SHA-256 hash
	UserAgent    string    `db:"user_agent"`
	IPAddress    string    `db:"ip_address"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"-"` // delivered only via httpOnly cookie
	ExpiresAt    time.Time `json:"expires_at"`
	User         SafeUser  `json:"user"`
}
