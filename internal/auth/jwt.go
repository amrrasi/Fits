package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/amrrasi/fits/internal/models"
)

const tokenIssuer = "fits-processor"

type JWTConfig struct {
	AccessSecret string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

// Claims: only the user id travels inside the JWT. Role/permissions are loaded fresh
// from the database by the auth middleware (Guard) and placed into the request context.
type Claims struct {
	UserID      int64       `json:"uid"`
	Email       string      `json:"-"`
	Role        models.Role `json:"-"`
	FullName    string      `json:"-"`
	Permissions []string    `json:"-"`
	jwt.RegisteredClaims
}

type TokenService struct{ cfg JWTConfig }

func NewTokenService(cfg JWTConfig) *TokenService { return &TokenService{cfg: cfg} }

func (ts *TokenService) IssueAccessToken(u *models.User) (string, time.Time, error) {
	exp := time.Now().Add(ts.cfg.AccessTTL)
	claims := Claims{
		UserID: u.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(ts.cfg.AccessSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, exp, nil
}

func (ts *TokenService) ValidateAccessToken(raw string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(ts.cfg.AccessSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(), jwt.WithLeeway(5*time.Second))
	if err != nil || !tok.Valid || claims.UserID <= 0 {
		return nil, errors.New("توکن نامعتبر است")
	}
	return claims, nil
}

// NewRefreshToken returns (plain, sha256-hex): 256 bits from crypto/rand.
func NewRefreshToken() (string, string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	plain := base64.RawURLEncoding.EncodeToString(b)
	return plain, HashRefreshToken(plain)
}

func HashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (ts *TokenService) RefreshTTL() time.Duration { return ts.cfg.RefreshTTL }
