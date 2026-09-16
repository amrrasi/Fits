package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
)

type contextKey string

const claimsKey contextKey = "claims"

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
				logger.S().Debugw("توکن احراز ناموفق بود", "err", err, "path", r.URL.Path)
				writeUnauthorized(w, "توکن نامعتبر یا باطل شده است.")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "متاسفانه احراز هویت نشدید")
				return
			}
			if !allowed[claims.Role] {
				writeForbidden(w, "پژوهشگر عزیز: متاسفانه نقش کاربری شما دسترسی کافی برای این عمل را ندارد")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission gates a route by one or more fine-grained permission
// codes (e.g. "files.delete"), resolved from the RBAC tables at login time
// and embedded in the access token. The request is allowed through if the
// caller holds AT LEAST ONE of the given codes.
func RequirePermission(codes ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(codes))
	for _, c := range codes {
		allowed[c] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "متاسفانه احراز هویت نشدید")
				return
			}
			for _, p := range claims.Permissions {
				if allowed[p] {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeForbidden(w, "دسترسی کافی برای این عمل را ندارید")
		})
	}
}

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

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
