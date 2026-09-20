package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
)

type contextKey string

const claimsKey contextKey = "claims"

func Middleware(ts *TokenService, guard *Guard) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearer(r)
			if tokenStr == "" {
				writeUnauthorized(w, "ابتدا وارد حساب کاربری خود شوید.")
				return
			}
			claims, err := ts.ValidateAccessToken(tokenStr)
			if err != nil {
				logger.S().Debugw("توکن احراز ناموفق بود", "err", err, "path", r.URL.Path)
				writeUnauthorized(w, "توکن نامعتبر یا منقضی شده است.")
				return
			}
			st, err := guard.Check(r.Context(), claims.UserID)
			if err != nil {
				if errors.Is(err, ErrAccountInactive) {
					writeUnauthorized(w, err.Error())
					return
				}
				logger.S().Errorw("guard check failed", "err", err)
				writeJSONError(w, http.StatusInternalServerError, "مشکلی در سرور پیش آمده است.")
				return
			}
			claims.Email, claims.Role, claims.FullName = st.User.Email, st.User.Role, st.User.FullName
			claims.Permissions = st.Perms
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
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

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]interface{}{"error": msg, "code": status})
	_, _ = w.Write(b)
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	writeJSONError(w, http.StatusUnauthorized, msg)
}
func writeForbidden(w http.ResponseWriter, msg string) { writeJSONError(w, http.StatusForbidden, msg) }
