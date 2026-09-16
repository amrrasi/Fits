package fits

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/config"
	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Processor ties together scanning, parsing, and persisting FITS files.
// It uses the split repositories and wraps each file ingestion in a transaction.
type Processor struct {
	cfg      config.FITSConfig
	pool     *pgxpool.Pool
	files    *repository.FileRepository
	headers  *repository.HeaderRepository
	metadata *repository.MetadataRepository
	jobs     *repository.JobRepository
}

// NewProcessor creates a Processor with the split repositories.
func NewProcessor(
	cfg config.FITSConfig,
	pool *pgxpool.Pool,
	files *repository.FileRepository,
	headers *repository.HeaderRepository,
	metadata *repository.MetadataRepository,
	jobs *repository.JobRepository,
) *Processor {
	return &Processor{
		cfg:      cfg,
		pool:     pool,
		files:    files,
		headers:  headers,
		metadata: metadata,
		jobs:     jobs,
	}
}

// Run scans ScanDir, processes every FITS file concurrently, and persists results.
// If a jobID > 0 is passed (triggered via API), that job row is reused;
// otherwise a new job is created.
func (p *Processor) Run(ctx context.Context) error {
	return p.RunWithJob(ctx, 0)
}

// RunWithJob runs the processor, using an existing job ID if provided (> 0).
// When existingJobID > 0 (i.e. triggered via POST /api/scan), the path to scan
// is read from that job's own scan_dir column — which may be a directory OR a
// single file path — NOT from the static FITS_SCAN_DIR config. This lets each
// scan request target whatever path/file the caller chose.
func (p *Processor) RunWithJob(ctx context.Context, existingJobID int64) error {
	start := time.Now()

	// ── Job record ────────────────────────────────────────────────────────────
	var jobID int64
	var scanTarget string
	if existingJobID > 0 {
		job, err := p.jobs.GetByID(ctx, existingJobID)
		if err != nil {
			return fmt.Errorf("processor: load job %d: %w", existingJobID, err)
		}
		jobID = job.ID
		scanTarget = job.ScanDir
	} else {
		scanTarget = p.cfg.ScanDir
		id, err := p.jobs.Create(ctx, scanTarget)
		if err != nil {
			return fmt.Errorf("processor: create job: %w", err)
		}
		jobID = id
	}

	log := logger.S().With("scan_dir", scanTarget, "job_id", jobID)
	log.Info("processor: starting")
	log.Infow("processor: job started")

	// ── Scan for files ────────────────────────────────────────────────────────
	// ScanDir also works when scanTarget is a single file path, not just a
	// directory — filepath.WalkDir visits a lone file exactly once.
	paths, err := ScanDir(scanTarget)
	if err != nil {
		msg := err.Error()
		_ = p.jobs.Finish(ctx, jobID, models.JobStatusFailed, 0, &msg)
		return fmt.Errorf("processor: scan: %w", err)
	}
	if len(paths) == 0 {
		log.Warn("processor: no FITS files found")
		_ = p.jobs.Finish(ctx, jobID, models.JobStatusCompleted, time.Since(start).Milliseconds(), nil)
		return nil
	}

	log.Infow("processor: files discovered", "count", len(paths))
	_ = p.jobs.UpdateProgress(ctx, jobID, len(paths), 0, 0, 0)

	// ── Worker pool ───────────────────────────────────────────────────────────
	pathCh := make(chan string, len(paths))
	for _, path := range paths {
		pathCh <- path
	}
	close(pathCh)

	var (
		wg           sync.WaitGroup
		doneAtomic   int64
		errorAtomic  int64
		dupAtomic    int64
	)

	workers := p.cfg.Workers
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wlog := log.With("worker", workerID)
			for path := range pathCh {
				if ctx.Err() != nil {
					return
				}
				status, procErr := p.processFile(ctx, jobID, path)
				switch {
				case procErr != nil:
					atomic.AddInt64(&errorAtomic, 1)
					wlog.Errorw("processor: file failed", "path", path, "err", procErr)
				case status == fileStatusDuplicate:
					atomic.AddInt64(&dupAtomic, 1)
				default:
					atomic.AddInt64(&doneAtomic, 1)
				}
				_ = p.jobs.UpdateProgress(ctx, jobID,
					len(paths),
					int(atomic.LoadInt64(&doneAtomic)),
					int(atomic.LoadInt64(&errorAtomic)),
					int(atomic.LoadInt64(&dupAtomic)),
				)
			}
		}(i)
	}

	wg.Wait()
	durationMs := time.Since(start).Milliseconds()

	// ── Determine final status ────────────────────────────────────────────────
	done := int(atomic.LoadInt64(&doneAtomic))
	errCount := int(atomic.LoadInt64(&errorAtomic))
	dupCount := int(atomic.LoadInt64(&dupAtomic))

	var finalStatus models.JobStatus
	switch {
	case ctx.Err() != nil:
		finalStatus = models.JobStatusCancelled
	case errCount == 0:
		finalStatus = models.JobStatusCompleted
	case done == 0 && dupCount == 0:
		finalStatus = models.JobStatusFailed
	default:
		finalStatus = models.JobStatusPartiallyFailed
	}

	_ = p.jobs.Finish(ctx, jobID, finalStatus, durationMs, nil)

	log.Infow("processor: done",
		"total", len(paths),
		"done", done,
		"duplicates", dupCount,
		"errors", errCount,
		"status", finalStatus,
		"duration_ms", durationMs,
	)
	return nil
}

// Result codes returned by processFile (only meaningful when err == nil).
const (
	fileStatusOK        = "ok"
	fileStatusDuplicate = "duplicate"
)

// processFile ingests one FITS file atomically in a single transaction.
// If anything fails, the transaction rolls back — no partial data.
// Returns fileStatusDuplicate (with nil error) if this file's content
// (checksum) already exists in the database under a different path.
func (p *Processor) processFile(ctx context.Context, jobID int64, path string) (string, error) {
	log := logger.S().With("path", path, "job_id", jobID)
	start := time.Now()

	// ── Parse (outside transaction — CPU work, no DB) ─────────────────────────
	result, err := ParseFile(path)
	if err != nil {
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FilePath: path,
			Stage:    "parse",
			Message:  err.Error(),
		})
		return "", fmt.Errorf("parse: %w", err)
	}

	// ── Skip unchanged files (exact same path, same content, already done) ──
	unchanged, err := p.files.PathDoneWithChecksum(ctx, path, result.File.Checksum)
	if err != nil {
		log.Warnw("processor: checksum check failed, processing anyway", "err", err)
	}
	if unchanged {
		log.Infow("processor: skipping unchanged file", "checksum", result.File.Checksum)
		return fileStatusOK, nil
	}

	// ── Detect duplicate content under a different path ──────────────────────
	dup, err := p.files.FindDoneByChecksum(ctx, result.File.Checksum, path)
	if err != nil {
		log.Warnw("processor: duplicate check failed, processing anyway", "err", err)
	}
	if dup != nil {
		reason := fmt.Sprintf("محتوای این فایل با «%s» (شناسه %d) یکسان است — چک‌سام تکراری", dup.FileName, dup.ID)
		result.File.Status = models.FileStatusSkipped
		fileID, upErr := p.files.UpsertFile(ctx, nil, &result.File)
		if upErr != nil {
			log.Warnw("processor: failed to record duplicate file row", "err", upErr)
			return "", fmt.Errorf("record duplicate: %w", upErr)
		}
		if err := p.files.MarkSkipped(ctx, fileID, reason); err != nil {
			log.Warnw("processor: failed to set skipped_reason", "err", err)
		}
		// Audit-trail row: which file was the original, which was the duplicate.
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FileID:   &fileID,
			FilePath: path,
			Stage:    "duplicate",
			Message:  reason,
		})
		log.Infow("processor: duplicate content detected",
			"original_path", dup.FilePath, "original_file_id", dup.ID)
		return fileStatusDuplicate, nil
	}

	// ── Atomic transaction ────────────────────────────────────────────────────
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("processor: begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// 1. Upsert file row
	fileID, err := p.files.UpsertFile(ctx, tx, &result.File)
	if err != nil {
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FilePath: path,
			Stage:    "insert",
			Message:  err.Error(),
		})
		return "", fmt.Errorf("upsert file: %w", err)
	}

	// 2. Delete old headers (idempotent reprocessing)
	if err := p.headers.DeleteByFileID(ctx, tx, fileID); err != nil {
		return "", fmt.Errorf("delete old headers: %w", err)
	}

	// 3. Bulk insert all headers
	if err := p.headers.BulkInsert(ctx, tx, fileID, result.Headers); err != nil {
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FileID:   &fileID,
			FilePath: path,
			Stage:    "insert",
			Message:  err.Error(),
		})
		return "", fmt.Errorf("insert headers: %w", err)
	}

	// 4. Upsert typed metadata
	result.Metadata.FileID = fileID
	if err := p.metadata.Upsert(ctx, tx, fileID, &result.Metadata); err != nil {
		log.Warnw("processor: metadata upsert failed (non-fatal)", "err", err)
		// Non-fatal: raw headers are the source of truth
	}

	// 5. Mark file done
	processingMs := time.Since(start).Milliseconds()
	if err := p.files.UpdateStatus(ctx, tx, fileID, models.FileStatusDone, nil, &processingMs); err != nil {
		return "", fmt.Errorf("update file status: %w", err)
	}

	// 6. Commit
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("processor: commit tx: %w", err)
	}
	tx = nil // prevent deferred rollback

	log.Infow("processor: file done",
		"file_id", fileID,
		"headers", len(result.Headers),
		"processing_ms", processingMs,
	)
	return fileStatusOK, nil
}

