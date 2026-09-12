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

// Service handles login, logout, and token refresh business logic.
type Service struct {
	users  *repository.UserRepository
	tokens *TokenService
}

// NewService creates an auth Service.
func NewService(users *repository.UserRepository, tokens *TokenService) *Service {
	return &Service{users: users, tokens: tokens}
}

// Login validates credentials and issues a token pair.
func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*models.TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Always return a generic error — don't reveal whether the email exists
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("auth: account is disabled")
	}

	if err := CheckPassword(password, user.PasswordHash); err != nil {
		logger.S().Warnw("auth: failed login attempt", "email", email, "ip", ip)
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	pair, err := s.issueTokens(ctx, user, userAgent, ip)
	if err != nil {
		return nil, err
	}

	// Stamp last login asynchronously — failure is non-fatal
	go func() {
		if err := s.users.UpdateLastLogin(context.Background(), user.ID); err != nil {
			logger.S().Warnw("auth: update last login failed", "user_id", user.ID, "err", err)
		}
	}()

	logger.S().Infow("auth: user logged in", "user_id", user.ID, "email", email, "ip", ip)
	return pair, nil
}

// Logout revokes the refresh token associated with the raw token string.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := HashToken(rawRefreshToken)
	if err := s.users.DeleteSession(ctx, hash); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	logger.S().Infow("auth: session revoked")
	return nil
}

// Refresh validates a refresh token and issues a new token pair (token rotation).
func (s *Service) Refresh(ctx context.Context, rawRefreshToken, userAgent, ip string) (*models.TokenPair, error) {
	hash := HashToken(rawRefreshToken)

	session, err := s.users.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid refresh token")
	}

	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		_ = s.users.DeleteSession(ctx, hash)
		return nil, fmt.Errorf("auth: refresh token expired")
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("auth: account is disabled")
	}

	// Rotate: delete old session, issue new pair
	_ = s.users.DeleteSession(ctx, hash)

	pair, err := s.issueTokens(ctx, user, userAgent, ip)
	if err != nil {
		return nil, err
	}

	logger.S().Infow("auth: token refreshed", "user_id", user.ID)
	return pair, nil
}

// issueTokens creates JWT pair + session row.
func (s *Service) issueTokens(ctx context.Context, user *models.User, userAgent, ip string) (*models.TokenPair, error) {
	accessToken, refreshToken, accessExp, err := s.tokens.IssueTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("auth: issue tokens: %w", err)
	}

	expiresAt := time.Now().Add(s.tokens.RefreshTTL())
	_, err = s.users.CreateSession(ctx, user.ID, HashToken(refreshToken), userAgent, ip, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
		User:         user.ToSafe(),
	}, nil
}

// ── Request helpers ───────────────────────────────────────────────────────────

// ClientIP extracts the real IP, respecting X-Forwarded-For.
func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}
