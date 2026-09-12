package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/amrrasi/fits/internal/models"
)

// JWTConfig holds signing secrets and expiry durations.
type JWTConfig struct {
	// AccessSecret is the HMAC-SHA256 signing key for access tokens.
	// Set via JWT_ACCESS_SECRET env var. Must be at least 32 chars in production.
	AccessSecret string

	// RefreshSecret is the signing key for refresh tokens.
	// Set via JWT_REFRESH_SECRET env var.
	RefreshSecret string

	// AccessTTL is how long access tokens are valid (default 15m).
	AccessTTL time.Duration

	// RefreshTTL is how long refresh tokens are valid (default 7d).
	RefreshTTL time.Duration
}

// Claims is the payload embedded in every access JWT.
type Claims struct {
	UserID   int64       `json:"uid"`
	Email    string      `json:"email"`
	Role     models.Role `json:"role"`
	FullName string      `json:"name"`
	jwt.RegisteredClaims
}

// TokenService handles issuing and validating JWTs.
type TokenService struct {
	cfg JWTConfig
}

// NewTokenService creates a TokenService with the given config.
func NewTokenService(cfg JWTConfig) *TokenService {
	if cfg.AccessTTL == 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}
	return &TokenService{cfg: cfg}
}

// IssueTokenPair creates a fresh access + refresh token pair for a user.
func (s *TokenService) IssueTokenPair(user *models.User) (accessToken, refreshToken string, accessExp time.Time, err error) {
	now := time.Now()
	accessExp = now.Add(s.cfg.AccessTTL)

	// ── Access token ──────────────────────────────────────────────────────────
	claims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		FullName: user.FullName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString([]byte(s.cfg.AccessSecret))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}

	// ── Refresh token — a random UUID (stored hashed in DB) ──────────────────
	refreshToken = uuid.New().String()
	return accessToken, refreshToken, accessExp, nil
}

// ValidateAccessToken parses and validates an access token, returning its claims.
func (s *TokenService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.AccessSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: invalid token claims")
	}
	return claims, nil
}

// RefreshTTL returns the refresh token lifetime for session expiry calculation.
func (s *TokenService) RefreshTTL() time.Duration {
	return s.cfg.RefreshTTL
}

// HashToken returns the SHA-256 hex of a raw token string.
// Refresh tokens are always stored hashed — never in plaintext.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
