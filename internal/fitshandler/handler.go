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

type Handler struct {
	svc     *fitsservice.Service
	scanDir string
	runFn   func(jobID int64)
}

func New(svc *fitsservice.Service, scanDir string) *Handler {
	return &Handler{
		svc:     svc,
		scanDir: scanDir,
		runFn:   func(_ int64) {},
	}
}

func (h *Handler) SetRunFn(fn func(jobID int64)) {
	h.runFn = fn
}

func (h *Handler) RegisterRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	editorMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	mux.Handle("GET /api/files", authMW(http.HandlerFunc(h.ListFiles)))
	mux.Handle("GET /api/files/{id}", authMW(http.HandlerFunc(h.GetFile)))
	mux.Handle("DELETE /api/files/{id}", authMW(adminMW(http.HandlerFunc(h.DeleteFile))))

	mux.Handle("GET /api/files/{id}/headers", authMW(http.HandlerFunc(h.ListHeaders)))

	mux.Handle("GET /api/files/{id}/metadata", authMW(http.HandlerFunc(h.GetMetadata)))
	mux.Handle("PUT /api/files/{id}/metadata", authMW(editorMW(http.HandlerFunc(h.EditMetadata))))
	mux.Handle("GET /api/files/{id}/metadata/history", authMW(http.HandlerFunc(h.GetMetadataHistory)))

	mux.Handle("GET /api/jobs", authMW(http.HandlerFunc(h.ListJobs)))
	mux.Handle("GET /api/jobs/{id}", authMW(http.HandlerFunc(h.GetJob)))
	mux.Handle("GET /api/jobs/{id}/errors", authMW(http.HandlerFunc(h.GetJobErrors)))
	mux.Handle("GET /api/jobs/{id}/status", authMW(http.HandlerFunc(h.GetJobStatus)))

	mux.Handle("POST /api/scan", authMW(adminMW(http.HandlerFunc(h.TriggerScan))))

	mux.Handle("GET /api/stats", authMW(http.HandlerFunc(h.Stats)))

	mux.HandleFunc("GET /ready", h.Ready)
}

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

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := api.PathID(r, "id")
	if err != nil {
		api.WriteBadRequest(w, err.Error())
		return
	}
	if err := h.svc.DeleteFile(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			api.WriteNotFound(w, "فایل یافت نشد")
			return
		}
		api.WriteInternalError(w, err)
		return
	}
	api.WriteNoContent(w)
}

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
		api.WriteBadRequest(w, err.Error())
		return
	}

	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, meta)
}

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

type triggerScanRequest struct {
	ScanDir string `json:"scan_dir"`
}

func (h *Handler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	var req triggerScanRequest
	_ = api.DecodeJSON(w, r, &req)

	scanDir := strings.TrimSpace(req.ScanDir)
	if scanDir == "" {
		scanDir = h.scanDir
	}
	if scanDir == "" {
		api.WriteBadRequest(w, "scan_dir الزامی است (یا FITS_SCAN_DIR را تنظیم کنید)")
		return
	}

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
		"message":  "اسکن شروع شد",
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ready","service":"fits-processor"}`))
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		api.WriteInternalError(w, err)
		return
	}
	api.WriteOK(w, stats)
}
