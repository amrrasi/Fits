package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/amrrasi/fits/internal/models"
)

func TestPasswordPolicy(t *testing.T) {
	bad := []string{"short1", "onlyletterslong", "1234567890123", "aaaaaaaaaaaa", "Admin@1234", "amirreza2024x"}
	for _, p := range bad {
		if ValidatePassword(p, "amirreza@x.com") == nil {
			t.Errorf("%q should be rejected", p)
		}
	}
	if err := ValidatePassword("Nebula-7-Orion-Dust", "a@b.com"); err != nil {
		t.Errorf("good password rejected: %v", err)
	}
	long := make([]byte, 80)
	for i := range long {
		long[i] = 'a' + byte(i%20)
	}
	if ValidatePassword(string(long)+"1", "") == nil {
		t.Error("passwords over 72 bytes must be rejected")
	}
}

func TestJWTRejectsForgeries(t *testing.T) {
	ts := NewTokenService(JWTConfig{AccessSecret: "0123456789abcdef0123456789abcdef", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	tok, _, err := ts.IssueAccessToken(&models.User{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := ts.ValidateAccessToken(tok); err != nil || c.UserID != 7 {
		t.Fatalf("valid token rejected: %v", err)
	}
	none := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{Issuer: tokenIssuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}})
	s, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := ts.ValidateAccessToken(s); err == nil {
		t.Error("alg=none must be rejected")
	}
	other := NewTokenService(JWTConfig{AccessSecret: "another-secret-another-secret-1234", AccessTTL: time.Minute})
	forged, _, _ := other.IssueAccessToken(&models.User{ID: 1})
	if _, err := ts.ValidateAccessToken(forged); err == nil {
		t.Error("wrong-secret token must be rejected")
	}
	exp := NewTokenService(JWTConfig{AccessSecret: "0123456789abcdef0123456789abcdef", AccessTTL: -time.Minute})
	old, _, _ := exp.IssueAccessToken(&models.User{ID: 1})
	if _, err := ts.ValidateAccessToken(old); err == nil {
		t.Error("expired token must be rejected")
	}
}

func TestLoginThrottle(t *testing.T) {
	th := NewLoginThrottle()
	for i := 0; i < 5; i++ {
		if _, b := th.Blocked("a@b.c", "1.1.1.1"); b {
			t.Fatal("blocked too early")
		}
		th.Fail("a@b.c", "1.1.1.1")
	}
	if _, b := th.Blocked("a@b.c", "1.1.1.1"); !b {
		t.Fatal("5 failures must block")
	}
	if _, b := th.Blocked("a@b.c", "2.2.2.2"); b {
		t.Fatal("a different IP must not be blocked for the same email (no victim lock-out)")
	}
}

func TestRefreshTokensAreUniqueAndHashed(t *testing.T) {
	a, ah := NewRefreshToken()
	b, _ := NewRefreshToken()
	if a == b || len(a) < 40 || HashRefreshToken(a) != ah || ah == a {
		t.Fatal("refresh token generation broken")
	}
}
