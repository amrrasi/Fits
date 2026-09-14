// Package userhandler provides HTTP handlers for user and role management.
package userhandler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
	"github.com/amrrasi/fits/internal/userservice"
)

// Handler exposes user management endpoints.
type Handler struct {
	svc *userservice.Service
}

// New creates a Handler.
func New(svc *userservice.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all user endpoints.
// authMW     — Bearer token validation middleware (applied to all routes)
// adminMW    — requires admin role
// editorMW  — requires admin or editor role
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW, adminMW func(http.Handler) http.Handler) {
	// ── /api/users/me — any authenticated user ────────────────────────────────
	mux.Handle("GET /api/users/me",
		authMW(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/users/me/password",
		authMW(http.HandlerFunc(h.ChangeMyPassword)))

	// ── /api/users — admin only ───────────────────────────────────────────────
	mux.Handle("GET /api/users",
		authMW(adminMW(http.HandlerFunc(h.List))))
	mux.Handle("POST /api/users",
		authMW(adminMW(http.HandlerFunc(h.Create))))
	mux.Handle("GET /api/users/{id}",
		authMW(adminMW(http.HandlerFunc(h.GetByID))))
	mux.Handle("PUT /api/users/{id}",
		authMW(adminMW(http.HandlerFunc(h.Update))))
	mux.Handle("DELETE /api/users/{id}",
		authMW(adminMW(http.HandlerFunc(h.Delete))))
	mux.Handle("PUT /api/users/{id}/password",
		authMW(adminMW(http.HandlerFunc(h.AdminResetPassword))))

	// ── /api/audit-logs — admin only ─────────────────────────────────────────
	mux.Handle("GET /api/audit-logs",
		authMW(adminMW(http.HandlerFunc(h.ListAuditLogs))))
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/users/me
// ─────────────────────────────────────────────────────────────────────────────

// Me returns the currently authenticated user's profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		api.WriteUnauthorized(w, "not authenticated")
		return
	}

	user, err := h.svc.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "user not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteOK(w, user.ToSafe())
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/users/me/password
// ─────────────────────────────────────────────────────────────────────────────

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangeMyPassword lets the logged-in user change their own password.
func (h *Handler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		api.WriteUnauthorized(w, "not authenticated")
		return
	}

	var req changePasswordRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		api.WriteBadRequest(w, "old_password and new_password are required")
		return
	}

	err := h.svc.ChangePassword(r.Context(), userservice.ChangePasswordInput{
		UserID:      claims.UserID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	api.WriteOK(w, map[string]string{"message": "password updated"})
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/users   (admin)
// ─────────────────────────────────────────────────────────────────────────────

// List returns a paginated, searchable list of all users.
//
// Query params:
//
//	page, page_size — pagination
//	search          — partial match on email or full_name
//	role            — filter by role (admin|editor|viewer)
//	active          — true|false
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, _ := api.Pagination(r)

	f := repository.ListUsersFilter{
		Search:   api.QueryString(r, "search", ""),
		Page:     page,
		PageSize: pageSize,
	}

	if role := api.QueryString(r, "role", ""); role != "" {
		f.Role = models.Role(role)
	}
	if active := r.URL.Query().Get("active"); active != "" {
		b := active == "true"
		f.IsActive = &b
	}

	result, err := h.svc.List(r.Context(), f)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}

	// Strip password hashes before sending
	safe := make([]models.SafeUser, len(result.Users))
	for i, u := range result.Users {
		safe[i] = u.ToSafe()
	}

	api.WritePaged(w, safe, result.Total, page, pageSize)
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/users   (admin)
// ─────────────────────────────────────────────────────────────────────────────

type createUserRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	FullName string      `json:"full_name"`
	Role     models.Role `json:"role"`
}

// Create adds a new user account.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		api.WriteBadRequest(w, "email and password are required")
		return
	}

	id, err := h.svc.Create(r.Context(), userservice.CreateInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, userservice.ErrEmailTaken) {
			api.WriteConflict(w, err.Error())
			return
		}
		api.WriteBadRequest(w, err.Error())
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}

	api.WriteCreated(w, user.ToSafe())
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/users/{id}   (admin)
// ─────────────────────────────────────────────────────────────────────────────

// GetByID returns a single user.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "user not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteOK(w, user.ToSafe())
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/users/{id}   (admin)
// ─────────────────────────────────────────────────────────────────────────────

type updateUserRequest struct {
	FullName string      `json:"full_name"`
	Role     models.Role `json:"role"`
	IsActive bool        `json:"is_active"`
}

// Update edits a user's full_name, role, and active status.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	var req updateUserRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}

	if err := h.svc.Update(r.Context(), id, userservice.UpdateInput{
		FullName: req.FullName,
		Role:     req.Role,
		IsActive: req.IsActive,
	}); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			api.WriteNotFound(w, "user not found")
		case errors.Is(err, userservice.ErrLastAdmin):
			api.WriteBadRequest(w, err.Error())
		default:
			api.WriteBadRequest(w, err.Error())
		}
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}

	api.WriteOK(w, user.ToSafe())
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE /api/users/{id}   (admin)
// ─────────────────────────────────────────────────────────────────────────────

// Delete permanently removes a user account.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	// Prevent self-deletion
	claims := auth.ClaimsFromContext(r.Context())
	if claims != nil && claims.UserID == id {
		api.WriteBadRequest(w, "you cannot delete your own account")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			api.WriteNotFound(w, "user not found")
		case errors.Is(err, userservice.ErrLastAdmin):
			api.WriteBadRequest(w, err.Error())
		default:
			api.WriteInternalError(w, err)
		}
		return
	}

	api.WriteNoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/users/{id}/password   (admin)
// ─────────────────────────────────────────────────────────────────────────────

type adminResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// AdminResetPassword lets an admin set a new password for any user.
func (h *Handler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	var req adminResetPasswordRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if req.NewPassword == "" {
		api.WriteBadRequest(w, "new_password is required")
		return
	}

	if err := h.svc.AdminResetPassword(r.Context(), id, req.NewPassword); err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	api.WriteOK(w, map[string]string{"message": "password reset"})
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/audit-logs   (admin only)
// ─────────────────────────────────────────────────────────────────────────────

// ListAuditLogs returns paginated audit log entries with optional filters.
// Query params: page, page_size, action, entity_type, date_from, date_to
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize, _ := api.Pagination(r)

	f := repository.ListAuditFilter{
		Action:     api.QueryString(r, "action", ""),
		EntityType: api.QueryString(r, "entity_type", ""),
		DateFrom:   api.QueryString(r, "date_from", ""),
		DateTo:     api.QueryString(r, "date_to", ""),
		Page:       page,
		PageSize:   pageSize,
	}

	logs, total, err := h.svc.ListAuditLogs(r.Context(), f)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if logs == nil {
		logs = []repository.AuditLog{}
	}
	api.WritePaged(w, logs, total, page, pageSize)
}
