package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const claimsKey contextKey = "claims"

// Middleware returns an HTTP middleware that validates the Bearer token
// and injects the parsed Claims into the request context.
// Returns 401 if the token is missing or invalid.
func Middleware(ts *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearer(r)
			if tokenStr == "" {
				writeUnauthorized(w, "missing authorization token")
				return
			}

			claims, err := ts.ValidateAccessToken(tokenStr)
			if err != nil {
				logger.S().Debugw("auth: token validation failed", "err", err, "path", r.URL.Path)
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that ensures the caller has at least the given role.
// Must be chained after Middleware.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "not authenticated")
				return
			}
			if !allowed[claims.Role] {
				writeForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext retrieves the JWT claims injected by Middleware.
// Returns nil if not present (unauthenticated request).
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

// ── helpers ───────────────────────────────────────────────────────────────────

func extractBearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `","code":401}`))
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"` + msg + `","code":403}`))
}
