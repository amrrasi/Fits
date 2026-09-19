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

type Processor struct {
	cfg      config.FITSConfig
	pool     *pgxpool.Pool
	files    *repository.FileRepository
	headers  *repository.HeaderRepository
	metadata *repository.MetadataRepository
	jobs     *repository.JobRepository
}

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

func (p *Processor) Run(ctx context.Context) error {
	return p.RunWithJob(ctx, 0)
}

func (p *Processor) RunWithJob(ctx context.Context, existingJobID int64) error {
	start := time.Now()

	var jobID int64
	var scanTarget string
	if existingJobID > 0 {
		job, err := p.jobs.GetByID(ctx, existingJobID)
		if err != nil {
			return fmt.Errorf("پردازشگر: load job %d: %w", existingJobID, err)
		}
		jobID = job.ID
		scanTarget = job.ScanDir
	} else {
		scanTarget = p.cfg.ScanDir
		id, err := p.jobs.Create(ctx, scanTarget)
		if err != nil {
			return fmt.Errorf("پردازشگر: create job: %w", err)
		}
		jobID = id
	}

	log := logger.S().With("scan_dir", scanTarget, "job_id", jobID)
	log.Info("پردازشگر: در حال شروع ...")
	log.Infow("پردازشگر: پردازش شروع شد")

	paths, err := ScanDir(scanTarget)
	if err != nil {
		msg := err.Error()
		_ = p.jobs.Finish(ctx, jobID, models.JobStatusFailed, 0, &msg)
		return fmt.Errorf("پردازشگر: scan: %w", err)
	}
	if len(paths) == 0 {
		log.Warn("پردازشگر: no FITS files found")
		_ = p.jobs.Finish(ctx, jobID, models.JobStatusCompleted, time.Since(start).Milliseconds(), nil)
		return nil
	}

	log.Infow("پردازشگر: files discovered", "count", len(paths))
	_ = p.jobs.UpdateProgress(ctx, jobID, len(paths), 0, 0, 0)

	pathCh := make(chan string, len(paths))
	for _, path := range paths {
		pathCh <- path
	}
	close(pathCh)

	var (
		wg          sync.WaitGroup
		doneAtomic  int64
		errorAtomic int64
		dupAtomic   int64
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
					wlog.Errorw("پردازشگر: file failed", "path", path, "err", procErr)
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

	log.Infow("پردازشگر: done",
		"total", len(paths),
		"done", done,
		"duplicates", dupCount,
		"errors", errCount,
		"status", finalStatus,
		"duration_ms", durationMs,
	)
	return nil
}

const (
	fileStatusOK        = "ok"
	fileStatusDuplicate = "duplicate"
)

func (p *Processor) processFile(ctx context.Context, jobID int64, path string) (string, error) {
	log := logger.S().With("path", path, "job_id", jobID)
	start := time.Now()

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

	unchanged, err := p.files.PathDoneWithChecksum(ctx, path, result.File.Checksum)
	if err != nil {
		log.Warnw("پردازشگر: checksum check failed, processing anyway", "err", err)
	}
	if unchanged {
		log.Infow("پردازشگر: skipping unchanged file", "checksum", result.File.Checksum)
		return fileStatusOK, nil
	}

	dup, err := p.files.FindDoneByChecksum(ctx, result.File.Checksum, path)
	if err != nil {
		log.Warnw("پردازشگر: duplicate check failed, processing anyway", "err", err)
	}
	if dup != nil {
		reason := fmt.Sprintf("محتوای این فایل با «%s» (شناسه %d) یکسان است — چک‌سام تکراری", dup.FileName, dup.ID)
		result.File.Status = models.FileStatusSkipped
		fileID, upErr := p.files.UpsertFile(ctx, nil, &result.File)
		if upErr != nil {
			log.Warnw("پردازشگر: failed to record duplicate file row", "err", upErr)
			return "", fmt.Errorf("record duplicate: %w", upErr)
		}
		if err := p.files.MarkSkipped(ctx, fileID, reason); err != nil {
			log.Warnw("پردازشگر: failed to set skipped_reason", "err", err)
		}
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FileID:   &fileID,
			FilePath: path,
			Stage:    "duplicate",
			Message:  reason,
		})
		log.Infow("پردازشگر: duplicate content detected",
			"original_path", dup.FilePath, "original_file_id", dup.ID)
		return fileStatusDuplicate, nil
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("پردازشگر: begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

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

	if err := p.headers.DeleteByFileID(ctx, tx, fileID); err != nil {
		return "", fmt.Errorf("delete old headers: %w", err)
	}

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

	result.Metadata.FileID = fileID
	if err := p.metadata.Upsert(ctx, tx, fileID, &result.Metadata); err != nil {
		log.Warnw("پردازشگر: metadata upsert failed (non-fatal)", "err", err)

	}

	processingMs := time.Since(start).Milliseconds()
	if err := p.files.UpdateStatus(ctx, tx, fileID, models.FileStatusDone, nil, &processingMs); err != nil {
		return "", fmt.Errorf("update file status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("پردازشگر: commit tx: %w", err)
	}
	tx = nil // prevent deferred rollback

	log.Infow("پردازشگر: file done",
		"file_id", fileID,
		"headers", len(result.Headers),
		"processing_ms", processingMs,
	)
	return fileStatusOK, nil
}
