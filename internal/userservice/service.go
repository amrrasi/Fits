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

type Service struct {
	repo *repository.UserRepository
}

func New(repo *repository.UserRepository) *Service {
	return &Service{repo: repo}
}


func (s *Service) List(ctx context.Context, f repository.ListUsersFilter) (*repository.ListUsersResult, error) {
	return s.repo.ListUsers(ctx, f)
}


func (s *Service) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u, nil
}

type CreateInput struct {
	Email    string
	Password string
	FullName string
	Role     models.Role
}

func (s *Service) Create(ctx context.Context, in CreateInput) (int64, error) {
	// Normalise
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.FullName = strings.TrimSpace(in.FullName)

	if in.Email == "" {
		return 0, fmt.Errorf("email is required")
	}
	if !strings.Contains(in.Email, "@") {
		return 0, fmt.Errorf("آدرس ایمیل نامعتبر")
	}
	if len(in.Password) < 8 {
		return 0, fmt.Errorf("رمز عبور شما باید بیش از 8 کاراکتر باشد")
	}
	if in.Role == "" {
		in.Role = models.RoleViewer
	}
	if !validRole(in.Role) {
		return 0, fmt.Errorf("نقش نامعتبر! نقش های مجاز: admin, editor, reader")
	}


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

type UpdateInput struct {
	FullName string
	Role     models.Role
	IsActive bool
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) error {
	in.FullName = strings.TrimSpace(in.FullName)

	if !validRole(in.Role) {
		return fmt.Errorf("نقش نامعتبر! نقش های مجاز: admin, editor, reader")
	}

	// Prevent locking out the last admin
	if in.Role != models.RoleAdmin || !in.IsActive {
		if err := s.guardLastAdmin(ctx, id); err != nil {
			return err
		}
	}

	return s.repo.UpdateUser(ctx, id, in.FullName, in.Role, in.IsActive)
}

type ChangePasswordInput struct {
	UserID      int64
	OldPassword string
	NewPassword string
}

func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if len(in.NewPassword) < 8 {
		return fmt.Errorf("رمز عبور جدید میبایست حداقل 8 کاراکتر باشد")
	}

	user, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("userservice: get user: %w", err)
	}

	if err := auth.CheckPassword(in.OldPassword, user.PasswordHash); err != nil {
		return fmt.Errorf("رمز عبور فعلی اشتباه میباشد")
	}

	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		return fmt.Errorf("userservice: hash password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, in.UserID, hash)
}

func (s *Service) AdminResetPassword(ctx context.Context, userID int64, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("رمز عبور میبایست حداقل 8 کاراکتر باشد")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("userservice: hash password: %w", err)
	}
	return s.repo.UpdatePassword(ctx, userID, hash)
}


func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.guardLastAdmin(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteUser(ctx, id)
}


func (s *Service) guardLastAdmin(ctx context.Context, userID int64) error {
	target, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if target.Role != models.RoleAdmin {
		return nil // not an admin, no guard needed
	}

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

var (
	ErrEmailTaken = fmt.Errorf("کاربری با ایمیل وارد شده وجود دارد")
	ErrLastAdmin  = fmt.Errorf("امکان حذف و یا تغییر آخرین فعالیت ادمین وجود ندارد")
)

// ListAuditLogs returns paginated audit log entries.
func (s *Service) ListAuditLogs(ctx context.Context, f repository.ListAuditFilter) ([]repository.AuditLog, int, error) {
	return s.repo.ListAuditLogs(ctx, f)
}
