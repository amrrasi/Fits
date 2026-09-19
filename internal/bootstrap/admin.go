// Package bootstrap makes sure a usable first administrator exists.
package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// EnsureAdmin creates (or re-activates) an admin when NO active admin exists.
// Password: ADMIN_PASSWORD from env, otherwise a random one printed ONCE to stderr (never logged to files).
func EnsureAdmin(ctx context.Context, users *repository.UserRepository, rbac *repository.RBACRepository, email, password string) error {
	n, err := users.CountActiveAdmins(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap: count admins: %w", err)
	}
	if n > 0 {
		return nil
	}
	generated := false
	if password == "" {
		b := make([]byte, 15)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		password = base64.RawURLEncoding.EncodeToString(b) + "9a"
		generated = true
	}
	if err := auth.ValidatePassword(password, email); err != nil {
		return fmt.Errorf("bootstrap: ADMIN_PASSWORD rejected: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	existing, err := users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		if err := users.UpdatePassword(ctx, existing.ID, hash); err != nil {
			return err
		}
		if err := users.UpdateUserGuarded(ctx, existing.ID, existing.FullName, models.RoleAdmin, true); err != nil {
			return err
		}
		_ = users.DeleteAllUserSessions(ctx, existing.ID)
		if err := rbac.SetPrimaryBuiltinRole(ctx, existing.ID, string(models.RoleAdmin)); err != nil {
			return err
		}
	case errors.Is(err, repository.ErrNotFound):
		id, err := users.CreateUser(ctx, email, hash, "مدیر سیستم", models.RoleAdmin)
		if err != nil {
			return err
		}
		if err := rbac.SetPrimaryBuiltinRole(ctx, id, string(models.RoleAdmin)); err != nil {
			return err
		}
	default:
		return err
	}
	logger.S().Infow("bootstrap: admin account is ready", "email", email)
	if generated {
		fmt.Fprintf(os.Stderr, "\n=====================================================\n"+
			"  ادمین اولیه ساخته شد — این رمز فقط یک‌بار نمایش داده می‌شود\n"+
			"  email:    %s\n  password: %s\n"+
			"  بعد از ورود، فوراً رمز را عوض کنید.\n"+
			"=====================================================\n\n", email, password)
	}
	return nil
}
