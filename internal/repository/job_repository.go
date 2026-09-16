package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/models"
)

// JobRepository handles processing_jobs and processing_errors.
type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

// Create inserts a new job row and returns its ID.
func (r *JobRepository) Create(ctx context.Context, scanDir string) (int64, error) {
	const q = `
		INSERT INTO processing_jobs (scan_dir, status, total_files, done_files, error_files, started_at)
		VALUES ($1, 'running', 0, 0, 0, NOW())
		RETURNING id`
	var id int64
	if err := r.pool.QueryRow(ctx, q, scanDir).Scan(&id); err != nil {
		return 0, fmt.Errorf("job_repo: create: %w", err)
	}
	return id, nil
}

// GetByID returns a single job.
func (r *JobRepository) GetByID(ctx context.Context, id int64) (*models.ProcessingJob, error) {
	const q = `
		SELECT id, scan_dir, status, total_files, done_files, error_files, duplicate_files,
		       started_at, finished_at, error_message
		FROM processing_jobs WHERE id=$1`

	j := &models.ProcessingJob{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&j.ID, &j.ScanDir, &j.Status,
		&j.TotalFiles, &j.DoneFiles, &j.ErrorFiles, &j.DuplicateFiles,
		&j.StartedAt, &j.FinishedAt, &j.ErrorMsg,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("job_repo: get by id: %w", err)
	}
	return j, nil
}

// UpdateProgress updates running counters.
func (r *JobRepository) UpdateProgress(ctx context.Context, jobID int64, total, done, errCount, duplicateCount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE processing_jobs SET total_files=$2, done_files=$3, error_files=$4, duplicate_files=$5 WHERE id=$1`,
		jobID, total, done, errCount, duplicateCount,
	)
	if err != nil {
		return fmt.Errorf("job_repo: update progress: %w", err)
	}
	return nil
}

// Finish marks a job terminal: completed | partially_failed | failed | cancelled.
func (r *JobRepository) Finish(ctx context.Context, jobID int64, status models.JobStatus, durationMs int64, errMsg *string) error {
	const q = `
		UPDATE processing_jobs
		SET status=$2, finished_at=NOW(), duration_ms=$3, error_message=$4
		WHERE id=$1`
	_, err := r.pool.Exec(ctx, q, jobID, string(status), durationMs, errMsg)
	if err != nil {
		return fmt.Errorf("job_repo: finish: %w", err)
	}
	return nil
}

// ListJobs returns a paginated list of jobs newest-first.
func (r *JobRepository) ListJobs(ctx context.Context, page, pageSize int) ([]models.ProcessingJob, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM processing_jobs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("job_repo: count: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, scan_dir, status, total_files, done_files, error_files, duplicate_files,
		       started_at, finished_at, error_message
		FROM processing_jobs
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("job_repo: list: %w", err)
	}
	defer rows.Close()

	var jobs []models.ProcessingJob
	for rows.Next() {
		var j models.ProcessingJob
		if err := rows.Scan(
			&j.ID, &j.ScanDir, &j.Status,
			&j.TotalFiles, &j.DoneFiles, &j.ErrorFiles, &j.DuplicateFiles,
			&j.StartedAt, &j.FinishedAt, &j.ErrorMsg,
		); err != nil {
			return nil, 0, fmt.Errorf("job_repo: scan: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

// InsertError records a per-file processing failure.
func (r *JobRepository) InsertError(ctx context.Context, e *models.ProcessingError) error {
	const q = `
		INSERT INTO processing_errors (job_id, file_id, file_path, stage, message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`
	_, err := r.pool.Exec(ctx, q, e.JobID, e.FileID, e.FilePath, e.Stage, e.Message)
	if err != nil {
		return fmt.Errorf("job_repo: insert error: %w", err)
	}
	return nil
}

// ListErrors returns all errors for a job.
func (r *JobRepository) ListErrors(ctx context.Context, jobID int64, page, pageSize int) ([]models.ProcessingError, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM processing_errors WHERE job_id=$1`, jobID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("job_repo: count errors: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, job_id, file_id, file_path, stage, message, created_at
		FROM processing_errors
		WHERE job_id=$1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`, jobID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("job_repo: list errors: %w", err)
	}
	defer rows.Close()

	var errs []models.ProcessingError
	for rows.Next() {
		var e models.ProcessingError
		if err := rows.Scan(&e.ID, &e.JobID, &e.FileID, &e.FilePath, &e.Stage, &e.Message, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("job_repo: scan error: %w", err)
		}
		errs = append(errs, e)
	}
	return errs, total, nil
}

// HasRunningJob returns true if any job is currently in 'running' status.
func (r *JobRepository) HasRunningJob(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM processing_jobs WHERE status='running')`,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("job_repo: check running: %w", err)
	}
	return exists, nil
}
