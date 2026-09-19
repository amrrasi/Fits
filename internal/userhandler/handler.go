// Package userhandler provides HTTP handlers for user and role management.
package userhandler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/audit"
	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
	"github.com/amrrasi/fits/internal/userservice"
)

// Handler exposes user management endpoints.
type Handler struct {
	svc   *userservice.Service
	audit *audit.Recorder
}

// New creates a Handler.
func New(svc *userservice.Service, rec *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: rec}
}

// fail maps service errors to safe, user-facing responses (never leaks internals).
func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		api.WriteNotFound(w, "مورد درخواستی پیدا نشد")
	case errors.Is(err, userservice.ErrEmailTaken):
		api.WriteConflict(w, err.Error())
	case errors.Is(err, userservice.ErrLastAdmin):
		api.WriteBadRequest(w, err.Error())
	default:
		api.WriteError(w, err)
	}
}

func actor(r *http.Request) *int64 {
	if c := auth.ClaimsFromContext(r.Context()); c != nil {
		id := c.UserID
		return &id
	}
	return nil
}

func idStr(id int64) string { return strconv.FormatInt(id, 10) }

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
		api.WriteUnauthorized(w, "ابتدا وارد حساب خود شوید")
		return
	}

	user, err := h.svc.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "کاربر پیدا نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	safe := user.ToSafe()
	safe.Permissions = claims.Permissions
	api.WriteOK(w, safe)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/users/me/password
// ─────────────────────────────────────────────────────────────────────────────

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		api.WriteUnauthorized(w, "ابتدا وارد حساب خود شوید")
		return
	}
	var req changePasswordRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		api.WriteBadRequest(w, "رمز عبور فعلی و جدید را وارد کنید")
		return
	}
	keep := ""
	if c, err := r.Cookie(auth.CookieName()); err == nil {
		keep = auth.HashRefreshToken(c.Value)
	}
	if err := h.svc.ChangePassword(r.Context(), userservice.ChangePasswordInput{
		UserID: claims.UserID, OldPassword: req.OldPassword, NewPassword: req.NewPassword, KeepSession: keep,
	}); err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "password.change", "user", idStr(claims.UserID), nil, nil)
	api.WriteOK(w, map[string]string{"message": "رمز عبور با موفقیت تغییر کرد؛ سایر دستگاه‌ها خارج شدند"})
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		api.WriteBadRequest(w, "ایمیل و رمز عبور را وارد کنید")
		return
	}
	id, err := h.svc.Create(r.Context(), userservice.CreateInput{
		Email: req.Email, Password: req.Password, FullName: req.FullName, Role: req.Role,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "user.create", "user", idStr(id), nil,
		map[string]interface{}{"email": user.Email, "full_name": user.FullName, "role": user.Role})
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
			api.WriteNotFound(w, "کاربر پیدا نشد")
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
	IsActive *bool       `json:"is_active"`
}

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
	if req.IsActive == nil {
		api.WriteBadRequest(w, "وضعیت فعال بودن حساب باید مشخص شود")
		return
	}
	old, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	var actorID int64
	if a := actor(r); a != nil {
		actorID = *a
	}
	if err := h.svc.Update(r.Context(), id, actorID, userservice.UpdateInput{
		FullName: req.FullName, Role: req.Role, IsActive: *req.IsActive,
	}); err != nil {
		h.fail(w, err)
		return
	}
	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "user.update", "user", idStr(id),
		map[string]interface{}{"full_name": old.FullName, "role": old.Role, "is_active": old.IsActive},
		map[string]interface{}{"full_name": user.FullName, "role": user.Role, "is_active": user.IsActive})
	api.WriteOK(w, user.ToSafe())
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE /api/users/{id}   (admin)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	if a := actor(r); a != nil && *a == id {
		api.WriteBadRequest(w, "نمی‌توانید حساب خودتان را حذف کنید")
		return
	}
	old, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "user.delete", "user", idStr(id),
		map[string]interface{}{"email": old.Email, "full_name": old.FullName, "role": old.Role}, nil)
	api.WriteNoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/users/{id}/password   (admin)
// ─────────────────────────────────────────────────────────────────────────────

type adminResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

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
		api.WriteBadRequest(w, "رمز عبور جدید را وارد کنید")
		return
	}
	if err := h.svc.AdminResetPassword(r.Context(), id, req.NewPassword); err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "password.reset", "user", idStr(id), nil, nil)
	api.WriteOK(w, map[string]string{"message": "رمز عبور کاربر بازنشانی شد و همه‌ی نشست‌های او بسته شد"})
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/audit-logs   (admin only)
// ─────────────────────────────────────────────────────────────────────────────

func parseDate(v string) (string, bool) {
	if v == "" {
		return "", true
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format(time.RFC3339), true
		}
	}
	return "", false
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize, _ := api.Pagination(r)
	from, ok1 := parseDate(api.QueryString(r, "date_from", ""))
	to, ok2 := parseDate(api.QueryString(r, "date_to", ""))
	if !ok1 || !ok2 {
		api.WriteBadRequest(w, "فرمت تاریخ نامعتبر است")
		return
	}
	f := repository.ListAuditFilter{
		Action:     api.QueryString(r, "action", ""),
		EntityType: api.QueryString(r, "entity_type", ""),
		DateFrom:   from,
		DateTo:     to,
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
		api.WriteBadRequest(w, "شناسه‌ی نقش الزامی است")
		return
	}
	var by int64
	if a := actor(r); a != nil {
		by = *a
	}
	if err := h.svc.AssignUserRole(r.Context(), userID, req.RoleID, by); err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "user.role_assign", "user", idStr(userID), nil, map[string]interface{}{"role_id": req.RoleID})
	api.WriteOK(w, map[string]string{"message": "نقش اختصاص داده شد"})
}

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
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "user.role_remove", "user", idStr(userID), map[string]interface{}{"role_id": roleID}, nil)
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

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	id, err := h.svc.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "role.create", "role", idStr(id), nil, map[string]interface{}{"name": req.Name})
	api.WriteCreated(w, map[string]interface{}{"id": id, "name": strings.ToLower(strings.TrimSpace(req.Name))})
}

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
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "role.delete", "role", idStr(id), nil, nil)
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
		h.fail(w, err)
		return
	}
	h.audit.Log(r, actor(r), "role.permissions", "role", idStr(roleID), nil, map[string]interface{}{"permissions": req.Permissions})
	api.WriteOK(w, map[string]string{"message": "دسترسی‌های نقش به‌روزرسانی شد"})
}
