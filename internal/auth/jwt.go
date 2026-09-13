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

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type Claims struct {
	UserID   int64       `json:"uid"`
	Email    string      `json:"email"`
	Role     models.Role `json:"role"`
	FullName string      `json:"name"`
	jwt.RegisteredClaims
}

type TokenService struct {
	cfg JWTConfig
}

func NewTokenService(cfg JWTConfig) *TokenService {
	if cfg.AccessTTL == 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 1 * 24 * time.Hour
	}
	return &TokenService{cfg: cfg}
}

func (s *TokenService) IssueTokenPair(user *models.User) (accessToken, refreshToken string, accessExp time.Time, err error) {
	now := time.Now()
	accessExp = now.Add(s.cfg.AccessTTL)

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

	refreshToken = uuid.New().String()
	return accessToken, refreshToken, accessExp, nil
}

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

func (s *TokenService) RefreshTTL() time.Duration {
	return s.cfg.RefreshTTL
}

func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
