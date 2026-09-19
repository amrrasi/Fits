package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/audit"
	"github.com/amrrasi/fits/internal/middleware"
)

const (
	refreshCookie = "fits_rt"
	csrfHeader    = "X-Requested-With"
	csrfValue     = "fits"
)

type Handler struct {
	svc          *Service
	audit        *audit.Recorder
	cookieSecure bool
}

func NewHandler(svc *Service, rec *audit.Recorder, cookieSecure bool) *Handler {
	return &Handler{svc: svc, audit: rec, cookieSecure: cookieSecure}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
	mux.Handle("POST /api/auth/logout-all", authMW(http.HandlerFunc(h.LogoutAll)))
}

// csrfOK: the refresh cookie is SameSite=Strict AND state-changing auth calls must carry a
// custom header, which a cross-site form/img request can never add.
func csrfOK(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(csrfHeader) != csrfValue {
		api.WriteJSON(w, http.StatusForbidden, api.ErrorResponse{Error: "درخواست نامعتبر است", Code: 403})
		return false
	}
	return true
}

func (h *Handler) setCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: token, Path: "/api", MaxAge: maxAge,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) writePair(w http.ResponseWriter, pair interface{}, refresh string, ttlSeconds int) {
	h.setCookie(w, refresh, ttlSeconds)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pair)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !csrfOK(w, r) {
		return
	}
	var req loginRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		api.WriteBadRequest(w, "ایمیل و رمز عبور را وارد کنید")
		return
	}
	ip := middleware.ClientIP(r)
	pair, err := h.svc.Login(r.Context(), req.Email, req.Password, r.UserAgent(), ip)
	if err != nil {
		var te *ThrottledError
		switch {
		case errors.As(err, &te):
			w.Header().Set("Retry-After", strconv.Itoa(int(te.RetryAfter.Seconds())+1))
			api.WriteJSON(w, http.StatusTooManyRequests, api.ErrorResponse{Error: te.Error(), Code: 429})
		case errors.Is(err, ErrInvalidCredentials):
			if len(req.Email) > 254 {
				req.Email = req.Email[:254]
			}
			h.audit.Log(r, nil, "user.login_failed", "user", req.Email, nil, nil)
			api.WriteJSON(w, http.StatusUnauthorized, api.ErrorResponse{Error: err.Error(), Code: 401})
		default:
			api.WriteInternalError(w, err)
		}
		return
	}
	uid := pair.User.ID
	h.audit.Log(r, &uid, "user.login", "user", strconv.FormatInt(uid, 10), nil, nil)
	h.writePair(w, pair, pair.RefreshToken, int(h.svc.tokens.RefreshTTL().Seconds()))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !csrfOK(w, r) {
		return
	}
	c, err := r.Cookie(refreshCookie)
	if err != nil || c.Value == "" {
		api.WriteJSON(w, http.StatusUnauthorized, api.ErrorResponse{Error: ErrInvalidRefresh.Error(), Code: 401})
		return
	}
	pair, err := h.svc.Refresh(r.Context(), c.Value, r.UserAgent(), middleware.ClientIP(r))
	if err != nil {
		h.setCookie(w, "", -1)
		if errors.Is(err, ErrInvalidRefresh) {
			api.WriteJSON(w, http.StatusUnauthorized, api.ErrorResponse{Error: err.Error(), Code: 401})
		} else {
			api.WriteInternalError(w, err)
		}
		return
	}
	h.writePair(w, pair, pair.RefreshToken, int(h.svc.tokens.RefreshTTL().Seconds()))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if !csrfOK(w, r) {
		return
	}
	if c, err := r.Cookie(refreshCookie); err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	h.setCookie(w, "", -1)
	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "با موفقیت خارج شدید"})
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	if !csrfOK(w, r) {
		return
	}
	claims := ClaimsFromContext(r.Context())
	if err := h.svc.LogoutAll(r.Context(), claims.UserID); err != nil {
		api.WriteInternalError(w, err)
		return
	}
	h.svc.guard.Invalidate(claims.UserID)
	h.audit.Log(r, &claims.UserID, "user.logout_all", "user", strconv.FormatInt(claims.UserID, 10), nil, nil)
	h.setCookie(w, "", -1)
	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "از همه‌ی دستگاه‌ها خارج شدید"})
}

// CookieName exposes the refresh cookie name to other packages (password change keeps this session).
func CookieName() string { return refreshCookie }
