// Package fitsservice contains business logic for FITS data operations.
// Handlers call this layer — never repositories directly.
package fitsservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Service is the FITS data service.
type Service struct {
	pool     *pgxpool.Pool
	files    *repository.FileRepository
	headers  *repository.HeaderRepository
	metadata *repository.MetadataRepository
	jobs     *repository.JobRepository
}

// New creates a Service wired to the given pool.
func New(
	pool *pgxpool.Pool,
	files *repository.FileRepository,
	headers *repository.HeaderRepository,
	metadata *repository.MetadataRepository,
	jobs *repository.JobRepository,
) *Service {
	return &Service{
		pool:     pool,
		files:    files,
		headers:  headers,
		metadata: metadata,
		jobs:     jobs,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Files
// ─────────────────────────────────────────────────────────────────────────────

// ListFiles returns a paginated, filtered list of FITS files.
func (s *Service) ListFiles(ctx context.Context, f repository.ListFilesFilter) (*repository.ListFilesResult, error) {
	return s.files.ListFiles(ctx, f)
}

// GetFile returns a single FITS file by ID.
func (s *Service) GetFile(ctx context.Context, id int64) (*models.FITSFile, error) {
	return s.files.GetByID(ctx, id)
}

// DeleteFile removes a FITS file and all related data (cascaded by DB).
// Admin only — enforced at the handler layer.
func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.files.Delete(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Headers
// ─────────────────────────────────────────────────────────────────────────────

// ListHeaders returns paginated raw headers for a file.
func (s *Service) ListHeaders(ctx context.Context, f repository.ListHeadersFilter) (*repository.ListHeadersResult, error) {
	// Verify file exists first
	if _, err := s.files.GetByID(ctx, f.FileID); err != nil {
		return nil, err
	}
	return s.headers.ListHeaders(ctx, f)
}

// ─────────────────────────────────────────────────────────────────────────────
// Metadata
// ─────────────────────────────────────────────────────────────────────────────

// GetMetadata returns the typed metadata for a file.
func (s *Service) GetMetadata(ctx context.Context, fileID int64) (*models.FITSMetadata, error) {
	if _, err := s.files.GetByID(ctx, fileID); err != nil {
		return nil, err
	}
	return s.metadata.GetByFileID(ctx, fileID)
}

// EditMetadataInput is the payload for a single field edit.
type EditMetadataInput struct {
	FileID    int64
	FieldName string
	NewValue  string
	Reason    string
	EditorID  int64
}

// EditMetadata applies one field edit to fits_metadata and records the override.
// Raw fits_headers is never touched.
func (s *Service) EditMetadata(ctx context.Context, in EditMetadataInput) error {
	in.FieldName = strings.ToLower(strings.TrimSpace(in.FieldName))

	// Validate field is editable
	allowed := repository.AllowedEditFields()
	if _, ok := allowed[in.FieldName]; !ok {
		return fmt.Errorf("field %q is not editable; allowed fields: %s",
			in.FieldName, joinKeys(allowed))
	}

	// Validate value makes sense for known numeric fields
	if err := validateMetadataValue(in.FieldName, in.NewValue); err != nil {
		return err
	}

	// Get current value for the override record
	current, err := s.metadata.GetByFileID(ctx, in.FileID)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("fitsservice: get current metadata: %w", err)
	}
	originalValue := currentFieldValue(current, in.FieldName)

	// Record the override (audit trail)
	reason := in.Reason
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	editorID := in.EditorID
	if err := s.metadata.InsertOverride(ctx, &repository.MetadataOverride{
		FileID:        in.FileID,
		FieldName:     in.FieldName,
		OriginalValue: originalValue,
		NewValue:      in.NewValue,
		Reason:        reasonPtr,
		EditedBy:      &editorID,
	}); err != nil {
		return fmt.Errorf("fitsservice: record override: %w", err)
	}

	// Apply the change
	if err := s.metadata.ApplyFieldUpdate(ctx, in.FileID, in.FieldName, in.NewValue); err != nil {
		return fmt.Errorf("fitsservice: apply field update: %w", err)
	}

	logger.S().Infow("fitsservice: metadata edited",
		"file_id", in.FileID,
		"field", in.FieldName,
		"old", originalValue,
		"new", in.NewValue,
		"editor", in.EditorID,
	)
	return nil
}

// GetOverrides returns the edit history for a file's metadata.
func (s *Service) GetOverrides(ctx context.Context, fileID int64) ([]repository.MetadataOverride, error) {
	return s.metadata.ListOverrides(ctx, fileID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs
// ─────────────────────────────────────────────────────────────────────────────

// ListJobs returns paginated processing jobs.
func (s *Service) ListJobs(ctx context.Context, page, pageSize int) ([]models.ProcessingJob, int, error) {
	return s.jobs.ListJobs(ctx, page, pageSize)
}

// GetJob returns a single job by ID.
func (s *Service) GetJob(ctx context.Context, id int64) (*models.ProcessingJob, error) {
	return s.jobs.GetByID(ctx, id)
}

// GetJobErrors returns errors for a job, paginated.
func (s *Service) GetJobErrors(ctx context.Context, jobID int64, page, pageSize int) ([]models.ProcessingError, int, error) {
	// Verify job exists
	if _, err := s.jobs.GetByID(ctx, jobID); err != nil {
		return nil, 0, err
	}
	return s.jobs.ListErrors(ctx, jobID, page, pageSize)
}

// TriggerScan starts a new scan job. Returns ErrScanAlreadyRunning if one is active.
// The actual scan runs in a goroutine — this returns the job ID immediately.
func (s *Service) TriggerScan(ctx context.Context, scanDir string, runFn func(jobID int64)) (int64, error) {
	running, err := s.jobs.HasRunningJob(ctx)
	if err != nil {
		return 0, fmt.Errorf("fitsservice: check running job: %w", err)
	}
	if running {
		return 0, ErrScanAlreadyRunning
	}

	jobID, err := s.jobs.Create(ctx, scanDir)
	if err != nil {
		return 0, fmt.Errorf("fitsservice: create job: %w", err)
	}

	// Run the scan in background — caller provides the run function
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
		defer cancel()
		runFn(jobID)
		_ = bgCtx
	}()

	logger.S().Infow("fitsservice: scan triggered", "job_id", jobID, "scan_dir", scanDir)
	return jobID, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// validateMetadataValue enforces basic constraints on editable fields.
func validateMetadataValue(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value for %q cannot be empty", field)
	}
	switch field {
	case "ra":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < 0 || f > 360 {
			return fmt.Errorf("ra must be a decimal number between 0 and 360")
		}
	case "dec":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < -90 || f > 90 {
			return fmt.Errorf("dec must be a decimal number between -90 and 90")
		}
	case "exptime", "gain", "rdnoise":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < 0 {
			return fmt.Errorf("%s must be a non-negative number", field)
		}
	case "airmass":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < 1 {
			return fmt.Errorf("airmass must be >= 1")
		}
	}
	return nil
}

// currentFieldValue reads the current string representation of a metadata field.
func currentFieldValue(m *models.FITSMetadata, field string) *string {
	if m == nil {
		return nil
	}
	var s string
	switch field {
	case "object":
		if m.Object != nil { s = *m.Object } else { return nil }
	case "observer":
		if m.Observer != nil { s = *m.Observer } else { return nil }
	case "telescop":
		if m.Telescop != nil { s = *m.Telescop } else { return nil }
	case "instrume":
		if m.Instrume != nil { s = *m.Instrume } else { return nil }
	case "filter":
		if m.Filter != nil { s = *m.Filter } else { return nil }
	case "ra":
		if m.RA != nil { s = fmt.Sprintf("%f", *m.RA) } else { return nil }
	case "dec":
		if m.Dec != nil { s = fmt.Sprintf("%f", *m.Dec) } else { return nil }
	case "exptime":
		if m.ExpTime != nil { s = fmt.Sprintf("%f", *m.ExpTime) } else { return nil }
	default:
		return nil
	}
	return &s
}

func joinKeys(m map[string]struct{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// ErrScanAlreadyRunning is returned when a scan is triggered while one is active.
var ErrScanAlreadyRunning = fmt.Errorf("a scan is already running")
