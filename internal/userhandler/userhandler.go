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

// RegisterRoutes wires all endpoints onto mux. authMW validates the bearer
// token; each admin-area route is additionally gated by its own fine-grained
// permission (resolved from the RBAC tables at login time).
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	perm := auth.RequirePermission

	// ── /api/users/me — any authenticated user ────────────────────────────────
	mux.Handle("GET /api/users/me",
		authMW(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/users/me/password",
		authMW(http.HandlerFunc(h.ChangeMyPassword)))

	// ── /api/users ─────────────────────────────────────────────────────────────
	mux.Handle("GET /api/users",
		authMW(perm("users.view")(http.HandlerFunc(h.List))))
	mux.Handle("POST /api/users",
		authMW(perm("users.create")(http.HandlerFunc(h.Create))))
	mux.Handle("GET /api/users/{id}",
		authMW(perm("users.view")(http.HandlerFunc(h.GetByID))))
	mux.Handle("PUT /api/users/{id}",
		authMW(perm("users.edit")(http.HandlerFunc(h.Update))))
	mux.Handle("DELETE /api/users/{id}",
		authMW(perm("users.delete")(http.HandlerFunc(h.Delete))))
	mux.Handle("PUT /api/users/{id}/password",
		authMW(perm("users.reset_password")(http.HandlerFunc(h.AdminResetPassword))))

	// ── /api/users/{id}/roles — extra (non-primary) role grants ───────────────
	mux.Handle("GET /api/users/{id}/roles",
		authMW(perm("users.view")(http.HandlerFunc(h.GetUserRoles))))
	mux.Handle("POST /api/users/{id}/roles",
		authMW(perm("roles.manage")(http.HandlerFunc(h.AssignUserRole))))
	mux.Handle("DELETE /api/users/{id}/roles/{roleId}",
		authMW(perm("roles.manage")(http.HandlerFunc(h.RemoveUserRole))))

	// ── /api/rbac — role & permission catalog management ──────────────────────
	mux.Handle("GET /api/rbac/roles",
		authMW(perm("roles.manage")(http.HandlerFunc(h.ListRoles))))
	mux.Handle("POST /api/rbac/roles",
		authMW(perm("roles.manage")(http.HandlerFunc(h.CreateRole))))
	mux.Handle("DELETE /api/rbac/roles/{id}",
		authMW(perm("roles.manage")(http.HandlerFunc(h.DeleteRole))))
	mux.Handle("PUT /api/rbac/roles/{id}/permissions",
		authMW(perm("roles.manage")(http.HandlerFunc(h.SetRolePermissions))))
	mux.Handle("GET /api/rbac/permissions",
		authMW(perm("roles.manage")(http.HandlerFunc(h.ListPermissions))))

	// ── /api/audit-logs ─────────────────────────────────────────────────────────
	mux.Handle("GET /api/audit-logs",
		authMW(perm("audit.view")(http.HandlerFunc(h.ListAuditLogs))))
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

// ─────────────────────────────────────────────────────────────────────────────
// RBAC — roles, permissions, and per-user role assignments
// ─────────────────────────────────────────────────────────────────────────────

// GetUserRoles returns every role currently assigned to a user.
func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	names, err := h.svc.GetUserRoleNames(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if names == nil {
		names = []string{}
	}
	api.WriteOK(w, map[string]interface{}{"roles": names})
}

type assignRoleRequest struct {
	RoleID int64 `json:"role_id"`
}

// AssignUserRole grants an additional role to a user, on top of whatever
// they already have (including their primary built-in role).
func (h *Handler) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	userID, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	var req assignRoleRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if req.RoleID < 1 {
		api.WriteBadRequest(w, "role_id الزامی است")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	var assignedBy int64
	if claims != nil {
		assignedBy = claims.UserID
	}

	if err := h.svc.AssignUserRole(r.Context(), userID, req.RoleID, assignedBy); err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, map[string]string{"message": "نقش اختصاص داده شد"})
}

// RemoveUserRole revokes a role from a user.
func (h *Handler) RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	userID, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	roleID, err := api.PathID(r, "roleId")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	if err := h.svc.RemoveUserRole(r.Context(), userID, roleID); err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteNoContent(w)
}

// ListRoles returns every role together with its permission codes.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if roles == nil {
		roles = []models.RoleWithPermissions{}
	}
	api.WriteOK(w, roles)
}

// ListPermissions returns the full permission catalog.
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if perms == nil {
		perms = []models.Permission{}
	}
	api.WriteOK(w, perms)
}

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateRole creates a new custom role (beyond admin/editor/viewer).
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	id, err := h.svc.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	api.WriteCreated(w, map[string]interface{}{"id": id, "name": req.Name})
}

// DeleteRole removes a custom role. Built-in roles cannot be deleted.
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	if err := h.svc.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "نقش یافت نشد (یا نقشی پیش‌فرض است و قابل حذف نیست)")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteNoContent(w)
}

type setRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// SetRolePermissions replaces a role's entire permission set.
func (h *Handler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	var req setRolePermissionsRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.SetRolePermissions(r.Context(), roleID, req.Permissions); err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, map[string]string{"message": "دسترسی‌های نقش به‌روزرسانی شد"})
}
