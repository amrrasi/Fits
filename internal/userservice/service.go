package userservice

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

var (
	ErrEmailTaken = errors.New("این ایمیل قبلاً ثبت شده است")
	ErrLastAdmin  = repository.ErrLastAdmin
	roleNameRe    = regexp.MustCompile(`^[a-z0-9_-]{2,32}$`)
)

// SessionInvalidator drops cached auth state so changes apply immediately.
type SessionInvalidator interface {
	Invalidate(userID int64)
	InvalidateAll()
}

type Service struct {
	repo *repository.UserRepository
	rbac *repository.RBACRepository
	inv  SessionInvalidator
}

func New(repo *repository.UserRepository, rbac *repository.RBACRepository, inv SessionInvalidator) *Service {
	return &Service{repo: repo, rbac: rbac, inv: inv}
}

func (s *Service) List(ctx context.Context, f repository.ListUsersFilter) (*repository.ListUsersResult, error) {
	return s.repo.ListUsers(ctx, f)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func validRole(r models.Role) bool {
	return r == models.RoleAdmin || r == models.RoleEditor || r == models.RoleViewer
}

func validateEmail(email string) error {
	if len(email) > 254 {
		return api.Invalid("ایمیل بیش از حد طولانی است")
	}
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return api.Invalid("ایمیل واردشده معتبر نیست")
	}
	return nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", api.Invalid("نام و نام خانوادگی الزامی است")
	}
	if len([]rune(name)) > 100 {
		return "", api.Invalid("نام نباید بیشتر از ۱۰۰ کاراکتر باشد")
	}
	for _, c := range name {
		if unicode.IsControl(c) {
			return "", api.Invalid("نام شامل کاراکتر نامعتبر است")
		}
	}
	return name, nil
}

type CreateInput struct {
	Email    string
	Password string
	FullName string
	Role     models.Role
}

func (s *Service) Create(ctx context.Context, in CreateInput) (int64, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if err := validateEmail(in.Email); err != nil {
		return 0, err
	}
	name, err := cleanName(in.FullName)
	if err != nil {
		return 0, err
	}
	if in.Role == "" {
		in.Role = models.RoleViewer
	}
	if !validRole(in.Role) {
		return 0, api.Invalid("نقش نامعتبر است (admin, editor, viewer)")
	}
	if err := auth.ValidatePassword(in.Password, in.Email); err != nil {
		return 0, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return 0, err
	}
	id, err := s.repo.CreateUser(ctx, in.Email, hash, name, in.Role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	if err := s.rbac.SetPrimaryBuiltinRole(ctx, id, string(in.Role)); err != nil {
		return 0, fmt.Errorf("userservice: assign role: %w", err)
	}
	return id, nil
}

type UpdateInput struct {
	FullName string
	Role     models.Role
	IsActive bool
}

// Update changes name/role/active. actorID protects against self-deactivation.
func (s *Service) Update(ctx context.Context, id, actorID int64, in UpdateInput) error {
	name, err := cleanName(in.FullName)
	if err != nil {
		return err
	}
	if !validRole(in.Role) {
		return api.Invalid("نقش نامعتبر است (admin, editor, viewer)")
	}
	if id == actorID && !in.IsActive {
		return api.Invalid("نمی‌توانید حساب خودتان را غیرفعال کنید")
	}
	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserGuarded(ctx, id, name, in.Role, in.IsActive); err != nil {
		return err
	}
	if err := s.rbac.SetPrimaryBuiltinRole(ctx, id, string(in.Role)); err != nil {
		return fmt.Errorf("userservice: sync role: %w", err)
	}
	if !in.IsActive || old.Role != in.Role {
		_ = s.repo.DeleteAllUserSessions(ctx, id) // force re-login with the new state
	}
	s.inv.Invalidate(id)
	return nil
}

type ChangePasswordInput struct {
	UserID      int64
	OldPassword string
	NewPassword string
	KeepSession string // sha256 of the caller's current refresh token (kept alive)
}

func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if auth.CheckPassword(in.OldPassword, u.PasswordHash) != nil {
		return api.Invalid("رمز عبور فعلی صحیح نیست")
	}
	if in.OldPassword == in.NewPassword {
		return api.Invalid("رمز عبور جدید باید با رمز فعلی متفاوت باشد")
	}
	if err := auth.ValidatePassword(in.NewPassword, u.Email); err != nil {
		return err
	}
	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, in.UserID, hash); err != nil {
		return err
	}
	// every OTHER device is signed out
	return s.repo.DeleteOtherSessions(ctx, in.UserID, in.KeepSession)
}

func (s *Service) AdminResetPassword(ctx context.Context, userID int64, newPassword string) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := auth.ValidatePassword(newPassword, u.Email); err != nil {
		return err
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	if err := s.repo.DeleteAllUserSessions(ctx, userID); err != nil {
		return err
	}
	s.inv.Invalidate(userID)
	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.DeleteUserGuarded(ctx, id); err != nil {
		return err
	}
	s.inv.Invalidate(id)
	return nil
}

func (s *Service) ListAuditLogs(ctx context.Context, f repository.ListAuditFilter) ([]repository.AuditLog, int, error) {
	return s.repo.ListAuditLogs(ctx, f)
}

func (s *Service) GetUserRoleNames(ctx context.Context, userID int64) ([]string, error) {
	return s.rbac.GetUserRoleNames(ctx, userID)
}

// AssignUserRole grants an EXTRA custom role. Built-in roles are managed via the user's role field.
func (s *Service) AssignUserRole(ctx context.Context, userID, roleID, assignedBy int64) error {
	if _, err := s.repo.GetByID(ctx, userID); err != nil {
		return err
	}
	role, err := s.rbac.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return api.Invalid("نقش‌های پیش‌فرض فقط از طریق ویرایش نقش کاربر قابل تغییر هستند")
	}
	if err := s.rbac.AssignUserRole(ctx, userID, roleID, &assignedBy); err != nil {
		return err
	}
	s.inv.Invalidate(userID)
	return nil
}

func (s *Service) RemoveUserRole(ctx context.Context, userID, roleID int64) error {
	role, err := s.rbac.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return api.Invalid("نقش‌های پیش‌فرض فقط از طریق ویرایش نقش کاربر قابل تغییر هستند")
	}
	if err := s.rbac.RemoveUserRole(ctx, userID, roleID); err != nil {
		return err
	}
	s.inv.Invalidate(userID)
	return nil
}

func (s *Service) ListRoles(ctx context.Context) ([]models.RoleWithPermissions, error) {
	return s.rbac.ListRoles(ctx)
}

func (s *Service) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	return s.rbac.ListPermissions(ctx)
}

func (s *Service) CreateRole(ctx context.Context, name, description string) (int64, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !roleNameRe.MatchString(name) {
		return 0, api.Invalid("نام نقش باید ۲ تا ۳۲ کاراکتر و فقط شامل حروف انگلیسی کوچک، عدد، خط تیره و زیرخط باشد")
	}
	if len([]rune(description)) > 200 {
		return 0, api.Invalid("توضیحات نباید بیشتر از ۲۰۰ کاراکتر باشد")
	}
	id, err := s.rbac.CreateRole(ctx, name, strings.TrimSpace(description))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, api.Invalid("نقشی با این نام از قبل وجود دارد")
		}
		return 0, err
	}
	return id, nil
}

func (s *Service) DeleteRole(ctx context.Context, roleID int64) error {
	if err := s.rbac.DeleteRole(ctx, roleID); err != nil {
		return err
	}
	s.inv.InvalidateAll()
	return nil
}

func (s *Service) SetRolePermissions(ctx context.Context, roleID int64, codes []string) error {
	role, err := s.rbac.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltin && role.Name == string(models.RoleAdmin) {
		return api.Invalid("دسترسی‌های نقش مدیر قابل تغییر نیست")
	}
	bad, err := s.rbac.UnknownPermissions(ctx, codes)
	if err != nil {
		return err
	}
	if len(bad) > 0 {
		return api.Invalid("کد دسترسی نامعتبر: " + strings.Join(bad, ", "))
	}
	if err := s.rbac.SetRolePermissions(ctx, roleID, codes); err != nil {
		return err
	}
	s.inv.InvalidateAll()
	return nil
}
