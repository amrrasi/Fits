package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/amrrasi/fits/internal/logger"
)

// ─────────────────────────────────────────────────────────────────────────────
// Standard response envelopes
// ─────────────────────────────────────────────────────────────────────────────

// Response is the standard success envelope.
type Response struct {
	Data interface{} `json:"data"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

// PagedResponse wraps a list result with pagination metadata.
type PagedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// Writers

func WriteJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.S().Warnw("خطا در نوشتن فایل Json!", "err", err)
	}
}

func WriteOK(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, Response{Data: data})
}

func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, Response{Data: data})
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WritePaged(w http.ResponseWriter, data interface{}, total, page, pageSize int) {
	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	WriteJSON(w, http.StatusOK, PagedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// Error writers

func WriteBadRequest(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: msg, Code: 400})
}

func WriteUnauthorized(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: msg, Code: 401})
}

func WriteForbidden(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: msg, Code: 403})
}

func WriteNotFound(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: msg, Code: 404})
}

func WriteConflict(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusConflict, ErrorResponse{Error: msg, Code: 409})
}

func WriteInternalError(w http.ResponseWriter, err error) {
	logger.S().Errorw("api: internal error", "err", err)
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "مشکلی در سرور داخلی پیش آمده", Code: 500})
}

// Request parsing helpers

// DecodeJSON reads and decodes a JSON request body into dst.
// Returns false and writes a 400 response if decoding fails.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		WriteBadRequest(w, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// PathID extracts a named segment from the URL path as int64.
// Uses the Go 1.22 pattern syntax: /api/users/{id}
func PathID(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, fmt.Errorf("missing path parameter: %s", name)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("نامعتبر %s: حتما باید به صورت عدد صحیح باشد", name)
	}
	if id < 1 {
		return 0, fmt.Errorf("نامعتبر %s: حتما باید مثبت باشد", name)
	}
	return id, nil
}

// QueryInt reads an integer query parameter, returning fallback if absent or invalid.
func QueryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 1 {
		return fallback
	}
	return i
}

// QueryString reads a string query parameter, returning fallback if absent.
func QueryString(r *http.Request, key, fallback string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return fallback
}

// Pagination reads page + page_size from query params with sensible defaults.
func Pagination(r *http.Request) (page, pageSize, offset int) {
	page = QueryInt(r, "page", 1)
	pageSize = QueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return
}
