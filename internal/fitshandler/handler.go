// Package fitshandler provides HTTP handlers for FITS data endpoints.
// All business logic lives in fitsservice — handlers only parse, call, respond.
package fitshandler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/auth"
	"github.com/amrrasi/fits/internal/fitsservice"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Handler exposes FITS data endpoints.
type Handler struct {
	svc     *fitsservice.Service
	scanDir string // default scan directory from config
}

// New creates a Handler.
func New(svc *fitsservice.Service, scanDir string) *Handler {
	return &Handler{svc: svc, scanDir: scanDir}
}

// RegisterRoutes wires all endpoints onto mux.
// authMW   — Bearer token required
// editorMW — admin or editor role required
// adminMW  — admin role required
func (h *Handler) RegisterRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	editorMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	// ── Files ─────────────────────────────────────────────────────────────────
	mux.Handle("GET /api/files",
		authMW(http.HandlerFunc(h.ListFiles)))
	mux.Handle("GET /api/files/{id}",
		authMW(http.HandlerFunc(h.GetFile)))
	mux.Handle("DELETE /api/files/{id}",
		authMW(adminMW(http.HandlerFunc(h.DeleteFile))))

	// ── Headers ───────────────────────────────────────────────────────────────
	mux.Handle("GET /api/files/{id}/headers",
		authMW(http.HandlerFunc(h.ListHeaders)))

	// ── Metadata ──────────────────────────────────────────────────────────────
	mux.Handle("GET /api/files/{id}/metadata",
		authMW(http.HandlerFunc(h.GetMetadata)))
	mux.Handle("PUT /api/files/{id}/metadata",
		authMW(editorMW(http.HandlerFunc(h.EditMetadata))))
	mux.Handle("GET /api/files/{id}/metadata/history",
		authMW(http.HandlerFunc(h.GetMetadataHistory)))

	// ── Jobs ──────────────────────────────────────────────────────────────────
	mux.Handle("GET /api/jobs",
		authMW(http.HandlerFunc(h.ListJobs)))
	mux.Handle("GET /api/jobs/{id}",
		authMW(http.HandlerFunc(h.GetJob)))
	mux.Handle("GET /api/jobs/{id}/errors",
		authMW(http.HandlerFunc(h.GetJobErrors)))
	mux.Handle("GET /api/jobs/{id}/status",
		authMW(http.HandlerFunc(h.GetJobStatus)))

	// ── Scan trigger ──────────────────────────────────────────────────────────
	mux.Handle("POST /api/scan",
		authMW(adminMW(http.HandlerFunc(h.TriggerScan))))

	// ── Readiness ─────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /ready", h.Ready)
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/files
// ─────────────────────────────────────────────────────────────────────────────

// ListFiles godoc
// Query params: page, page_size, search, status, sort, order, date_from, date_to
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

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/files/{id}
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	file, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "file not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteOK(w, file)
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE /api/files/{id}    (admin only)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	if err := h.svc.DeleteFile(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "file not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteNoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/files/{id}/headers
// ─────────────────────────────────────────────────────────────────────────────

// ListHeaders godoc
// Query params: page, page_size, search, hdu_index
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
	if hduStr := r.URL.Query().Get("hdu_index"); hduStr != "" {
		if hduIdx := api.QueryInt(r, "hdu_index", -1); hduIdx >= 0 {
			f.HDUIndex = &hduIdx
		}
	}

	result, err := h.svc.ListHeaders(r.Context(), f)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "file not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WritePaged(w, result.Headers, result.Total, page, pageSize)
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/files/{id}/metadata
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "metadata not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteOK(w, meta)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/files/{id}/metadata    (editor or admin)
// ─────────────────────────────────────────────────────────────────────────────

type editMetadataRequest struct {
	FieldName string `json:"field_name"` // e.g. "object", "ra", "filter"
	NewValue  string `json:"new_value"`
	Reason    string `json:"reason"` // optional
}

func (h *Handler) EditMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		api.WriteUnauthorized(w, "not authenticated")
		return
	}

	var req editMetadataRequest
	if !api.DecodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.FieldName) == "" || strings.TrimSpace(req.NewValue) == "" {
		api.WriteBadRequest(w, "field_name and new_value are required")
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
			api.WriteNotFound(w, "file or metadata not found")
			return
		}
		api.WriteBadRequest(w, err.Error())
		return
	}

	// Return updated metadata
	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, meta)
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/files/{id}/metadata/history
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/jobs
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/jobs/{id}
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "job not found")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, job)
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/jobs/{id}/errors
// ─────────────────────────────────────────────────────────────────────────────

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
			api.WriteNotFound(w, "job not found")
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

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/jobs/{id}/status   — lightweight poll endpoint
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "job not found")
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

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/scan    (admin only)
// ─────────────────────────────────────────────────────────────────────────────

type triggerScanRequest struct {
	ScanDir string `json:"scan_dir"` // optional — falls back to FITS_SCAN_DIR
}

func (h *Handler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	var req triggerScanRequest
	// Body is optional — ignore decode errors
	_ = api.DecodeJSON(w, r, &req)

	scanDir := strings.TrimSpace(req.ScanDir)
	if scanDir == "" {
		scanDir = h.scanDir
	}
	if scanDir == "" {
		api.WriteBadRequest(w, "scan_dir is required (or set FITS_SCAN_DIR)")
		return
	}

	// The run function is nil here — the actual processor will be injected via SetRunFn
	// For now return job ID; the processor goroutine is started by the service
	jobID, err := h.svc.TriggerScan(r.Context(), scanDir, h.runFn)
	if err != nil {
		if errors.Is(err, fitsservice.ErrScanAlreadyRunning) {
			api.WriteConflict(w, err.Error())
			return
		}
		api.WriteInternalError(w, err)
		return
	}

	api.WriteCreated(w, map[string]interface{}{
		"job_id":   jobID,
		"scan_dir": scanDir,
		"message":  "scan started",
	})
}

// runFn is set by SetRunFn — it is the actual scan-and-process function.
var _ = (*Handler)(nil) // ensure interface

func (h *Handler) runFn(_ int64) {
	// Default no-op — replaced via SetRunFn from main
}

// SetRunFn injects the actual processor run function.
// Called from main after both the handler and processor are initialised.
func (h *Handler) SetRunFn(fn func(jobID int64)) {
	h.runFn = fn
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /ready
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ready","service":"fits-processor"}`))
}
