package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
)

// Handler exposes auth HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates an auth Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires auth endpoints onto a ServeMux.
// Prefix should be "/api/auth".
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
}

// ── POST /api/auth/login ──────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login godoc
// @Summary     Authenticate user
// @Description Returns access + refresh token pair on valid credentials
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body loginRequest true "Credentials"
// @Success     200 {object} models.TokenPair
// @Failure     400 {object} errorResponse
// @Failure     401 {object} errorResponse
// @Router      /api/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body", 400})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{"email and password are required", 400})
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Email, req.Password,
		r.UserAgent(), ClientIP(r))
	if err != nil {
		logger.S().Infow("auth: login failed", "email", req.Email, "err", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse{err.Error(), 401})
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// ── POST /api/auth/logout ─────────────────────────────────────────────────────

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout godoc
// @Summary     Revoke refresh token
// @Description Invalidates the supplied refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body logoutRequest true "Refresh token"
// @Success     200 {object} map[string]string
// @Failure     400 {object} errorResponse
// @Router      /api/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body", 400})
		return
	}
	if req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{"refresh_token is required", 400})
		return
	}

	// Best-effort — don't expose errors to caller
	_ = h.svc.Logout(r.Context(), req.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// ── POST /api/auth/refresh ────────────────────────────────────────────────────

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh godoc
// @Summary     Refresh access token
// @Description Issues a new token pair, rotating the refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body refreshRequest true "Refresh token"
// @Success     200 {object} models.TokenPair
// @Failure     401 {object} errorResponse
// @Router      /api/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body", 400})
		return
	}
	if req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{"refresh_token is required", 400})
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken,
		r.UserAgent(), ClientIP(r))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{err.Error(), 401})
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// ── shared helpers ────────────────────────────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.S().Warnw("auth: write response failed", "err", err)
	}
}
