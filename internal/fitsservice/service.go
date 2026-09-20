package fitsservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/api"
	"github.com/amrrasi/fits/internal/fits"
	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

type Service struct {
	pool     *pgxpool.Pool
	files    *repository.FileRepository
	headers  *repository.HeaderRepository
	metadata *repository.MetadataRepository
	jobs     *repository.JobRepository

	scanRoot string
	scanner  Scanner
}

type Scanner interface {
	Start(scanDir string) (int64, error)
}

func (s *Service) SetScanner(sc Scanner) { s.scanner = sc }

func (s *Service) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func New(
	pool *pgxpool.Pool,
	files *repository.FileRepository,
	headers *repository.HeaderRepository,
	metadata *repository.MetadataRepository,
	jobs *repository.JobRepository,
	scanRoot string,
) *Service {
	return &Service{
		scanRoot: scanRoot,
		pool:     pool,
		files:    files,
		headers:  headers,
		metadata: metadata,
		jobs:     jobs,
	}
}

func (s *Service) ListFiles(ctx context.Context, f repository.ListFilesFilter) (*repository.ListFilesResult, error) {
	return s.files.ListFiles(ctx, f)
}

func (s *Service) GetFile(ctx context.Context, id int64) (*models.FITSFile, error) {
	return s.files.GetByID(ctx, id)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.files.Delete(ctx, id)
}

func (s *Service) ListHeaders(ctx context.Context, f repository.ListHeadersFilter) (*repository.ListHeadersResult, error) {
	if _, err := s.files.GetByID(ctx, f.FileID); err != nil {
		return nil, err
	}
	return s.headers.ListHeaders(ctx, f)
}

func (s *Service) GetMetadata(ctx context.Context, fileID int64) (*models.FITSMetadata, error) {
	if _, err := s.files.GetByID(ctx, fileID); err != nil {
		return nil, err
	}
	return s.metadata.GetByFileID(ctx, fileID)
}

type EditMetadataInput struct {
	FileID    int64
	FieldName string
	NewValue  string
	Reason    string
	EditorID  int64
}

func (s *Service) EditMetadata(ctx context.Context, in EditMetadataInput) error {
	in.FieldName = strings.ToLower(strings.TrimSpace(in.FieldName))
	if _, ok := repository.AllowedEditFields()[in.FieldName]; !ok {
		return api.Invalid("این فیلد قابل ویرایش نیست")
	}
	in.NewValue = strings.TrimSpace(in.NewValue)
	in.Reason = strings.TrimSpace(in.Reason)
	if len([]rune(in.Reason)) > 500 {
		return api.Invalid("توضیح تغییر نباید بیشتر از ۵۰۰ کاراکتر باشد")
	}
	if err := validateMetadataValue(in.FieldName, in.NewValue); err != nil {
		return err
	}
	current, err := s.metadata.GetByFileID(ctx, in.FileID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("fitsservice: get current metadata: %w", err)
	}
	var reasonPtr *string
	if in.Reason != "" {
		r := in.Reason
		reasonPtr = &r
	}
	editorID := in.EditorID
	if err := s.metadata.EditFieldTx(ctx, &repository.MetadataOverride{
		FileID: in.FileID, FieldName: in.FieldName, OriginalValue: currentFieldValue(current, in.FieldName),
		NewValue: in.NewValue, Reason: reasonPtr, EditedBy: &editorID,
	}); err != nil {
		return err
	}
	logger.S().Infow("fitsservice: metadata edited", "file_id", in.FileID, "field", in.FieldName, "editor", in.EditorID)
	return nil
}

func (s *Service) GetOverrides(ctx context.Context, fileID int64) ([]repository.MetadataOverride, error) {
	return s.metadata.ListOverrides(ctx, fileID)
}

func (s *Service) ListJobs(ctx context.Context, page, pageSize int) ([]models.ProcessingJob, int, error) {
	return s.jobs.ListJobs(ctx, page, pageSize)
}

func (s *Service) GetJob(ctx context.Context, id int64) (*models.ProcessingJob, error) {
	return s.jobs.GetByID(ctx, id)
}

func (s *Service) GetJobErrors(ctx context.Context, jobID int64, page, pageSize int) ([]models.ProcessingError, int, error) {
	if _, err := s.jobs.GetByID(ctx, jobID); err != nil {
		return nil, 0, err
	}
	return s.jobs.ListErrors(ctx, jobID, page, pageSize)
}

func (s *Service) TriggerScan(_ context.Context, requested string) (int64, string, error) {
	dir, err := fits.ResolveScanDir(s.scanRoot, requested)
	if err != nil {
		return 0, "", api.Invalid(err.Error())
	}
	if s.scanner == nil {
		return 0, "", errors.New("fitsservice: scanner not configured")
	}
	jobID, err := s.scanner.Start(dir)
	if err != nil {
		return 0, "", err
	}
	logger.S().Infow("fitsservice: scan triggered", "job_id", jobID, "scan_dir", dir)
	return jobID, dir, nil
}

func numRange(field, value string, min, max float64) error {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < min || f > max {
		return api.Invalid(fmt.Sprintf("مقدار «%s» باید عددی بین %v و %v باشد", field, min, max))
	}
	return nil
}

func validateMetadataValue(field, value string) error {
	if value == "" {
		return api.Invalid("مقدار جدید نمی‌تواند خالی باشد")
	}
	if len([]rune(value)) > 200 {
		return api.Invalid("مقدار نباید بیشتر از ۲۰۰ کاراکتر باشد")
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return api.Invalid("مقدار شامل کاراکتر نامعتبر است")
		}
	}
	switch field {
	case "ra":
		return numRange(field, value, 0, 360)
	case "dec", "site_lat":
		return numRange(field, value, -90, 90)
	case "site_lon":
		return numRange(field, value, -180, 180)
	case "site_elev":
		return numRange(field, value, -500, 10000)
	case "exptime", "gain", "rdnoise":
		return numRange(field, value, 0, 1e7)
	case "airmass":
		return numRange(field, value, 1, 100)
	case "set_temp", "ccd_temp", "amb_temp":
		return numRange(field, value, -273.15, 200)
	case "date_obs":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return api.Invalid("تاریخ باید به شکل YYYY-MM-DD باشد")
		}
	case "time_obs":
		if _, err := time.Parse("15:04:05", value); err != nil {
			return api.Invalid("زمان باید به شکل HH:MM:SS باشد")
		}
	}
	return nil
}

func currentFieldValue(m *models.FITSMetadata, field string) *string {
	if m == nil {
		return nil
	}
	var s string
	switch field {
	case "object":
		if m.Object != nil {
			s = *m.Object
		} else {
			return nil
		}
	case "observer":
		if m.Observer != nil {
			s = *m.Observer
		} else {
			return nil
		}
	case "telescop":
		if m.Telescop != nil {
			s = *m.Telescop
		} else {
			return nil
		}
	case "instrume":
		if m.Instrume != nil {
			s = *m.Instrume
		} else {
			return nil
		}
	case "filter":
		if m.Filter != nil {
			s = *m.Filter
		} else {
			return nil
		}
	case "ra":
		if m.RA != nil {
			s = fmt.Sprintf("%f", *m.RA)
		} else {
			return nil
		}
	case "dec":
		if m.Dec != nil {
			s = fmt.Sprintf("%f", *m.Dec)
		} else {
			return nil
		}
	case "exptime":
		if m.ExpTime != nil {
			s = fmt.Sprintf("%f", *m.ExpTime)
		} else {
			return nil
		}
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

var ErrScanAlreadyRunning = fits.ErrScanAlreadyRunning

type Stats struct {
	TotalFiles   int `json:"total_files"`
	DoneFiles    int `json:"done_files"`
	ErrorFiles   int `json:"error_files"`
	PendingFiles int `json:"pending_files"`
	TotalJobs    int `json:"total_jobs"`
	RunningJobs  int `json:"running_jobs"`
	TotalHeaders int `json:"total_headers"`
}

func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	rows, err := s.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM fits_files GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("fitsservice: stats files: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats.TotalFiles += count
		switch status {
		case "done":
			stats.DoneFiles = count
		case "error":
			stats.ErrorFiles = count
		case "pending":
			stats.PendingFiles = count
		}
	}

	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM processing_jobs`).Scan(&stats.TotalJobs)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM processing_jobs WHERE status='running'`).Scan(&stats.RunningJobs)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM fits_headers`).Scan(&stats.TotalHeaders)

	return stats, nil
}
