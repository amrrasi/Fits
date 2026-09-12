package models

import (
	"time"
)

// Role constants — match the user_role PostgreSQL ENUM.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// CanEdit returns true for roles that may mutate FITS data.
func (r Role) CanEdit() bool {
	return r == RoleAdmin || r == RoleEditor
}

// CanAdmin returns true only for the admin role.
func (r Role) CanAdmin() bool {
	return r == RoleAdmin
}

// User is the domain model for an authenticated user.
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

// SafeUser is a User without the password hash — safe to send over the wire.
type SafeUser struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Role        Role       `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ToSafe strips the password hash.
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

// Session tracks a single issued refresh token.
type Session struct {
	ID           string    `db:"id"`
	UserID       int64     `db:"user_id"`
	RefreshToken string    `db:"refresh_token"` // stored as SHA-256 hash
	UserAgent    string    `db:"user_agent"`
	IPAddress    string    `db:"ip_address"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

// TokenPair is what the login/refresh endpoints return to the client.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         SafeUser  `json:"user"`
}
