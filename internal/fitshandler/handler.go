// Package fitshandler provides HTTP handlers for FITS data endpoints.
// All business logic lives in fitsservice — handlers only parse, call, respond.
package fitshandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/audit"
	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/fitsservice"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Handler exposes FITS data endpoints.
type Handler struct {
	svc   *fitsservice.Service
	audit *audit.Recorder
}

// New creates a Handler.
func New(svc *fitsservice.Service, rec *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: rec}
}

func actor(r *http.Request) *int64 {
	if c := auth.ClaimsFromContext(r.Context()); c != nil {
		id := c.UserID
		return &id
	}
	return nil
}

// RegisterRoutes wires all endpoints onto mux. authMW validates the bearer
// token; each route is additionally gated by its own fine-grained permission
// (resolved from the RBAC tables at login time — see internal/auth).
func (h *Handler) RegisterRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
) {
	perm := auth.RequirePermission

	// Files
	mux.Handle("GET /api/files", authMW(perm("files.view")(http.HandlerFunc(h.ListFiles))))
	mux.Handle("GET /api/files/{id}", authMW(perm("files.view")(http.HandlerFunc(h.GetFile))))
	mux.Handle("DELETE /api/files/{id}", authMW(perm("files.delete")(http.HandlerFunc(h.DeleteFile))))

	// Headers
	mux.Handle("GET /api/files/{id}/headers", authMW(perm("files.view")(http.HandlerFunc(h.ListHeaders))))

	// Metadata
	mux.Handle("GET /api/files/{id}/metadata", authMW(perm("files.view")(http.HandlerFunc(h.GetMetadata))))
	mux.Handle("PUT /api/files/{id}/metadata", authMW(perm("files.metadata.edit")(http.HandlerFunc(h.EditMetadata))))
	mux.Handle("GET /api/files/{id}/metadata/history", authMW(perm("files.view")(http.HandlerFunc(h.GetMetadataHistory))))

	// Jobs
	mux.Handle("GET /api/jobs", authMW(perm("jobs.view")(http.HandlerFunc(h.ListJobs))))
	mux.Handle("GET /api/jobs/{id}", authMW(perm("jobs.view")(http.HandlerFunc(h.GetJob))))
	mux.Handle("GET /api/jobs/{id}/errors", authMW(perm("jobs.view")(http.HandlerFunc(h.GetJobErrors))))
	mux.Handle("GET /api/jobs/{id}/status", authMW(perm("jobs.view")(http.HandlerFunc(h.GetJobStatus))))

	// Scan trigger
	mux.Handle("POST /api/scan", authMW(perm("files.scan")(http.HandlerFunc(h.TriggerScan))))

	// Stats (dashboard) — anyone who can view files can see the summary
	mux.Handle("GET /api/stats", authMW(perm("files.view")(http.HandlerFunc(h.Stats))))

	// Readiness
	mux.HandleFunc("GET /ready", h.Ready)
}

// ── GET /api/files ────────────────────────────────────────────────────────────

func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	page, pageSize, _ := api.Pagination(r)

	f := repository.ListFilesFilter{
		Search:    api.QueryString(r, "search", ""),
		SortBy:    api.QueryString(r, "sort", "created_at"),
		SortOrder: api.QueryString(r, "order", "desc"),
		Page:      page,
		PageSize:  pageSize,
	}
	if s := r.URL.Query().Get("status"); s != "" {
		f.Status = models.FileStatus(s)
	}
	if v := r.URL.Query().Get("date_from"); v != "" {
		if t, err := time.Parse(time.DateOnly, v); err == nil {
			f.DateFrom = &t
		}
	}
	if v := r.URL.Query().Get("date_to"); v != "" {
		if t, err := time.Parse(time.DateOnly, v); err == nil {
			t = t.Add(24*time.Hour - time.Second)
			f.DateTo = &t
		}
	}

	result, err := h.svc.ListFiles(r.Context(), f)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WritePaged(w, result.Files, result.Total, page, pageSize)
}

// ── GET /api/files/{id} ───────────────────────────────────────────────────────

func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	file, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "فایل یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, file)
}

// ── DELETE /api/files/{id} (admin) ───────────────────────────────────────────

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	old, gerr := h.svc.GetFile(r.Context(), id)
	if err := h.svc.DeleteFile(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "فایل یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	var oldVal interface{}
	if gerr == nil {
		oldVal = map[string]interface{}{"file_name": old.FileName, "file_path": old.FilePath}
	}
	h.audit.Log(r, actor(r), "file.delete", "fits_file", strconv.FormatInt(id, 10), oldVal, nil)
	api.WriteNoContent(w)
}

// ── GET /api/files/{id}/headers ───────────────────────────────────────────────

func (h *Handler) ListHeaders(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	page, pageSize, _ := api.Pagination(r)
	f := repository.ListHeadersFilter{
		FileID:   id,
		Search:   api.QueryString(r, "search", ""),
		Page:     page,
		PageSize: pageSize,
	}
	if hduIdx := api.QueryInt(r, "hdu_index", -1); hduIdx >= 0 {
		f.HDUIndex = &hduIdx
	}

	result, err := h.svc.ListHeaders(r.Context(), f)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "فایل یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WritePaged(w, result.Headers, result.Total, page, pageSize)
}

// ── GET /api/files/{id}/metadata ─────────────────────────────────────────────

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "متادیتا یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, meta)
}

// ── PUT /api/files/{id}/metadata (editor/admin) ───────────────────────────────

type editMetadataRequest struct {
	FieldName string `json:"field_name"`
	NewValue  string `json:"new_value"`
	Reason    string `json:"reason"`
}

func (h *Handler) EditMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		api.WriteUnauthorized(w, "احراز هویت نشدید")
		return
	}
	var req editMetadataRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.FieldName) == "" || strings.TrimSpace(req.NewValue) == "" {
		api.WriteBadRequest(w, "field_name و new_value الزامی هستند")
		return
	}

	if err := h.svc.EditMetadata(r.Context(), fitsservice.EditMetadataInput{
		FileID:    id,
		FieldName: req.FieldName,
		NewValue:  req.NewValue,
		Reason:    req.Reason,
		EditorID:  claims.UserID,
	}); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "فایل یا متادیتا یافت نشد")
			return
		}
		api.WriteError(w, err)
		return
	}
	h.audit.Log(r, actor(r), "metadata.edit", "fits_file", strconv.FormatInt(id, 10), nil,
		map[string]interface{}{"field": strings.ToLower(strings.TrimSpace(req.FieldName)), "new_value": strings.TrimSpace(req.NewValue), "reason": req.Reason})

	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, meta)
}

// ── GET /api/files/{id}/metadata/history ─────────────────────────────────────

func (h *Handler) GetMetadataHistory(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	overrides, err := h.svc.GetOverrides(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if overrides == nil {
		overrides = []repository.MetadataOverride{}
	}
	api.WriteOK(w, overrides)
}

// ── GET /api/jobs ─────────────────────────────────────────────────────────────

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	page, pageSize, _ := api.Pagination(r)
	jobs, total, err := h.svc.ListJobs(r.Context(), page, pageSize)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	if jobs == nil {
		jobs = []models.ProcessingJob{}
	}
	api.WritePaged(w, jobs, total, page, pageSize)
}

// ── GET /api/jobs/{id} ────────────────────────────────────────────────────────

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "جاب یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, job)
}

// ── GET /api/jobs/{id}/errors ─────────────────────────────────────────────────

func (h *Handler) GetJobErrors(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	page, pageSize, _ := api.Pagination(r)
	errs, total, err := h.svc.GetJobErrors(r.Context(), id, page, pageSize)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "جاب یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	if errs == nil {
		errs = []models.ProcessingError{}
	}
	api.WritePaged(w, errs, total, page, pageSize)
}

// ── GET /api/jobs/{id}/status ─────────────────────────────────────────────────

func (h *Handler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "جاب یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, map[string]interface{}{
		"id":          job.ID,
		"status":      job.Status,
		"total_files": job.TotalFiles,
		"done_files":  job.DoneFiles,
		"error_files": job.ErrorFiles,
		"finished_at": job.FinishedAt,
	})
}

// ── POST /api/scan (admin) ────────────────────────────────────────────────────

type triggerScanRequest struct {
	ScanDir string `json:"scan_dir"`
}

func (h *Handler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	var req triggerScanRequest
	if !api.DecodeJSONOptional(w, r, &req) { // body is optional
		return
	}
	jobID, dir, err := h.svc.TriggerScan(r.Context(), req.ScanDir)
	if err != nil {
		if errors.Is(err, fitsservice.ErrScanAlreadyRunning) {
			api.WriteConflict(w, err.Error())
			return
		}
		api.WriteError(w, err)
		return
	}
	h.audit.Log(r, actor(r), "scan.trigger", "scan", strconv.FormatInt(jobID, 10), nil, map[string]interface{}{"scan_dir": dir})
	api.WriteCreated(w, map[string]interface{}{
		"job_id":   jobID,
		"scan_dir": dir,
		"message":  "اسکن شروع شد",
	})
}

// ── GET /ready ────────────────────────────────────────────────────────────────

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")
	if err := h.svc.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	_, _ = w.Write([]byte(`{"status":"ready","service":"fits-processor"}`))
}

// ── GET /api/stats ────────────────────────────────────────────────────────────
// Returns summary counts for the dashboard. Public to all authenticated users.

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, stats)
}
