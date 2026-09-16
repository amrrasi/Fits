package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

type Service struct {
	users  *repository.UserRepository
	rbac   *repository.RBACRepository
	tokens *TokenService
}

func NewService(users *repository.UserRepository, rbac *repository.RBACRepository, tokens *TokenService) *Service {
	return &Service{users: users, rbac: rbac, tokens: tokens}
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*models.TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Always return a generic error — don't reveal whether the email exists
		return nil, fmt.Errorf("خطایی در ورود رخ داده")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("حساب کاربری شما در حال حاضر فعال نمیباشد! ")
	}

	if err := CheckPassword(password, user.PasswordHash); err != nil {
		logger.S().Warnw("تلاش ناموفق برای ورود به حساب کاربری", "email", email, "ip", ip)
		return nil, fmt.Errorf(" خطای نامشخص! ")
	}

	pair, err := s.issueTokens(ctx, user, userAgent, ip)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := s.users.UpdateLastLogin(context.Background(), user.ID); err != nil {
			logger.S().Warnw("آپدیت آخرین ورود با خطا مواجه شده است", "user_id", user.ID, "err", err)
		}
	}()

	logger.S().Infow("با موفقیت وارد حساب کاربری شدید ...", "user_id", user.ID, "email", email, "ip", ip)
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := HashToken(rawRefreshToken)
	if err := s.users.DeleteSession(ctx, hash); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	logger.S().Infow("نشست شما باطل شد")
	return nil
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken, userAgent, ip string) (*models.TokenPair, error) {
	hash := HashToken(rawRefreshToken)

	session, err := s.users.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("توکن به‌روز رسانی نامعتبر")
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.users.DeleteSession(ctx, hash)
		return nil, fmt.Errorf("نشست شما باطل شده")
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("کاربر یافت نشد")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("حساب کاربری غیرفعال شده است")
	}

	_ = s.users.DeleteSession(ctx, hash)

	pair, err := s.issueTokens(ctx, user, userAgent, ip)
	if err != nil {
		return nil, err
	}

	logger.S().Infow("توکن رفرش شد", "user_id", user.ID)
	return pair, nil
}

func (s *Service) issueTokens(ctx context.Context, user *models.User, userAgent, ip string) (*models.TokenPair, error) {
	perms, err := s.rbac.GetUserPermissionCodes(ctx, user.ID)
	if err != nil {
		logger.S().Warnw("auth: failed to resolve permissions, issuing token with none", "user_id", user.ID, "err", err)
		perms = []string{}
	}

	accessToken, refreshToken, accessExp, err := s.tokens.IssueTokenPair(user, perms)
	if err != nil {
		return nil, fmt.Errorf("خطایی رخ داده است: %w", err)
	}

	expiresAt := time.Now().Add(s.tokens.RefreshTTL())
	_, err = s.users.CreateSession(ctx, user.ID, HashToken(refreshToken), userAgent, ip, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("نشست ساخته شد: %w", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
		User:         user.ToSafe(),
	}, nil
}

func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}
