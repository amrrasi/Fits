package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

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

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

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

	_ = h.svc.Logout(r.Context(), req.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

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
