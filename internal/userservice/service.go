// Package userservice contains business logic for user and role management.
package userservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Service handles user management operations.
type Service struct {
	repo *repository.UserRepository
}

// New creates a user Service.
func New(repo *repository.UserRepository) *Service {
	return &Service{repo: repo}
}

// ─────────────────────────────────────────────────────────────────────────────
// Read
// ─────────────────────────────────────────────────────────────────────────────

// List returns a filtered, paginated list of users.
func (s *Service) List(ctx context.Context, f repository.ListUsersFilter) (*repository.ListUsersResult, error) {
	return s.repo.ListUsers(ctx, f)
}

// GetByID returns a single user by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────────────────────

// CreateInput is the payload for creating a new user.
type CreateInput struct {
	Email    string
	Password string
	FullName string
	Role     models.Role
}

// Create validates and inserts a new user. Returns the new user's ID.
func (s *Service) Create(ctx context.Context, in CreateInput) (int64, error) {
	// Normalise
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.FullName = strings.TrimSpace(in.FullName)

	// Validate
	if in.Email == "" {
		return 0, fmt.Errorf("email is required")
	}
	if !strings.Contains(in.Email, "@") {
		return 0, fmt.Errorf("invalid email address")
	}
	if len(in.Password) < 8 {
		return 0, fmt.Errorf("password must be at least 8 characters")
	}
	if in.Role == "" {
		in.Role = models.RoleViewer
	}
	if !validRole(in.Role) {
		return 0, fmt.Errorf("invalid role: must be admin, editor, or viewer")
	}

	// Duplicate check
	exists, err := s.repo.EmailExists(ctx, in.Email)
	if err != nil {
		return 0, fmt.Errorf("userservice: check email: %w", err)
	}
	if exists {
		return 0, ErrEmailTaken
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return 0, fmt.Errorf("userservice: hash password: %w", err)
	}

	id, err := s.repo.CreateUser(ctx, in.Email, hash, in.FullName, in.Role)
	if err != nil {
		return 0, fmt.Errorf("userservice: create: %w", err)
	}
	return id, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────────────────

// UpdateInput is the payload for editing a user.
type UpdateInput struct {
	FullName string
	Role     models.Role
	IsActive bool
}

// Update edits a user's profile fields and role.
// Callers must verify the caller is admin before calling this.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) error {
	in.FullName = strings.TrimSpace(in.FullName)

	if !validRole(in.Role) {
		return fmt.Errorf("invalid role: must be admin, editor, or viewer")
	}

	// Prevent locking out the last admin
	if in.Role != models.RoleAdmin || !in.IsActive {
		if err := s.guardLastAdmin(ctx, id); err != nil {
			return err
		}
	}

	return s.repo.UpdateUser(ctx, id, in.FullName, in.Role, in.IsActive)
}

// ─────────────────────────────────────────────────────────────────────────────
// Password
// ─────────────────────────────────────────────────────────────────────────────

// ChangePasswordInput is the payload for changing a user's own password.
type ChangePasswordInput struct {
	UserID      int64
	OldPassword string
	NewPassword string
}

// ChangePassword verifies the old password then sets the new one.
func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if len(in.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	user, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("userservice: get user: %w", err)
	}

	if err := auth.CheckPassword(in.OldPassword, user.PasswordHash); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		return fmt.Errorf("userservice: hash password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, in.UserID, hash)
}

// AdminResetPassword lets an admin set a new password for any user without
// needing the old one.
func (s *Service) AdminResetPassword(ctx context.Context, userID int64, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("userservice: hash password: %w", err)
	}
	return s.repo.UpdatePassword(ctx, userID, hash)
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

// Delete hard-deletes a user. Prevents deleting the last admin.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.guardLastAdmin(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteUser(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// guardLastAdmin returns an error if deleting/demoting userID would leave no admins.
func (s *Service) guardLastAdmin(ctx context.Context, userID int64) error {
	target, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if target.Role != models.RoleAdmin {
		return nil // not an admin, no guard needed
	}

	// Count active admins
	t := true
	result, err := s.repo.ListUsers(ctx, repository.ListUsersFilter{
		Role:     models.RoleAdmin,
		IsActive: &t,
		Page:     1,
		PageSize: 2,
	})
	if err != nil {
		return fmt.Errorf("userservice: count admins: %w", err)
	}
	if result.Total <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func validRole(r models.Role) bool {
	return r == models.RoleAdmin || r == models.RoleEditor || r == models.RoleViewer
}

// ─────────────────────────────────────────────────────────────────────────────
// Sentinel errors
// ─────────────────────────────────────────────────────────────────────────────

var (
	ErrEmailTaken = fmt.Errorf("a user with that email already exists")
	ErrLastAdmin  = fmt.Errorf("cannot remove or demote the last active admin account")
)
