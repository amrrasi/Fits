package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

const maxSessionsPerUser = 10

// ErrInvalidCredentials is deliberately the ONLY login failure message (no user enumeration).
var (
	ErrInvalidCredentials = errors.New("ایمیل یا رمز عبور صحیح نیست")
	ErrInvalidRefresh     = errors.New("نشست شما منقضی شده است؛ دوباره وارد شوید")
)

type ThrottledError struct{ RetryAfter time.Duration }

func (e *ThrottledError) Error() string {
	return "تلاش‌های ناموفق بیش از حد بوده است؛ چند دقیقه دیگر دوباره تلاش کنید"
}

type Service struct {
	users    *repository.UserRepository
	rbac     *repository.RBACRepository
	tokens   *TokenService
	guard    *Guard
	throttle *LoginThrottle
}

func NewService(users *repository.UserRepository, rbac *repository.RBACRepository, tokens *TokenService, guard *Guard) *Service {
	return &Service{users: users, rbac: rbac, tokens: tokens, guard: guard, throttle: NewLoginThrottle()}
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*models.TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 || len(password) > 1024 {
		return nil, ErrInvalidCredentials
	}
	if d, blocked := s.throttle.Blocked(email, ip); blocked {
		return nil, &ThrottledError{RetryAfter: d}
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("auth: login lookup: %w", err)
		}
		burnPasswordCheck(password) // same cost as a real check
		s.throttle.Fail(email, ip)
		logger.S().Warnw("ورود ناموفق", "email", email, "ip", ip)
		return nil, ErrInvalidCredentials
	}
	// password first, THEN active check: an outsider can't learn that an account is disabled
	if CheckPassword(password, user.PasswordHash) != nil || !user.IsActive {
		s.throttle.Fail(email, ip)
		logger.S().Warnw("ورود ناموفق", "email", email, "ip", ip)
		return nil, ErrInvalidCredentials
	}
	s.throttle.Success(email, ip)

	pair, err := s.issueTokens(ctx, user, userAgent, ip)
	if err != nil {
		return nil, err
	}
	go func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.users.UpdateLastLogin(c, user.ID); err != nil {
			logger.S().Warnw("آپدیت آخرین ورود ناموفق بود", "user_id", user.ID, "err", err)
		}
	}()
	logger.S().Infow("ورود موفق", "user_id", user.ID, "ip", ip)
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	return s.users.DeleteSession(ctx, HashRefreshToken(rawRefreshToken))
}

// LogoutAll revokes every session of the user.
func (s *Service) LogoutAll(ctx context.Context, userID int64) error {
	if err := s.users.DeleteAllUserSessions(ctx, userID); err != nil {
		return err
	}
	return nil
}

func (s *Service) Refresh(ctx context.Context, raw, userAgent, ip string) (*models.TokenPair, error) {
	if raw == "" {
		return nil, ErrInvalidRefresh
	}
	// single-use: atomically consume, so a token can never be replayed concurrently
	session, err := s.users.ConsumeSession(ctx, HashRefreshToken(raw))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrInvalidRefresh
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil || !user.IsActive {
		return nil, ErrInvalidRefresh
	}
	return s.issueTokens(ctx, user, userAgent, ip)
}

func (s *Service) issueTokens(ctx context.Context, user *models.User, userAgent, ip string) (*models.TokenPair, error) {
	perms, err := s.rbac.GetUserPermissionCodes(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: resolve permissions: %w", err)
	}
	access, exp, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return nil, err
	}
	plain, hash := NewRefreshToken()
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	if _, err = s.users.CreateSession(ctx, user.ID, hash, userAgent, ip, time.Now().Add(s.tokens.RefreshTTL())); err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}
	if err := s.users.TrimSessions(ctx, user.ID, maxSessionsPerUser); err != nil {
		logger.S().Warnw("trim sessions failed", "err", err)
	}
	safe := user.ToSafe()
	safe.Permissions = perms
	return &models.TokenPair{AccessToken: access, RefreshToken: plain, ExpiresAt: exp, User: safe}, nil
}
