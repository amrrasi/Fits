package fits

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/amrrasi/fits/internal/config"
	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
	"github.com/amrrasi/fits/internal/repository"
)

// Processor ties together scanning, parsing, and persisting FITS files.
type Processor struct {
	cfg  config.FITSConfig
	repo *repository.FITSRepository
}

// NewProcessor creates a Processor.
func NewProcessor(cfg config.FITSConfig, repo *repository.FITSRepository) *Processor {
	return &Processor{cfg: cfg, repo: repo}
}

// Run scans ScanDir, processes every FITS file with cfg.Workers goroutines,
// and persists results to the database.  It returns after all files are done.
func (p *Processor) Run(ctx context.Context) error {
	log := logger.S().With("scan_dir", p.cfg.ScanDir, "workers", p.cfg.Workers)
	log.Info("processor: starting")

	// ── Create job record ──────────────────────────────────────────────────────
	jobID, err := p.repo.CreateJob(ctx, p.cfg.ScanDir)
	if err != nil {
		return fmt.Errorf("processor: %w", err)
	}
	log.Infow("processor: job created", "job_id", jobID)

	// ── Scan for files ─────────────────────────────────────────────────────────
	paths, err := ScanDir(p.cfg.ScanDir)
	if err != nil {
		msg := err.Error()
		_ = p.repo.FinishJob(ctx, jobID, models.JobStatusFailed, &msg)
		return fmt.Errorf("processor: scan: %w", err)
	}
	if len(paths) == 0 {
		log.Warn("processor: no FITS files found")
		_ = p.repo.FinishJob(ctx, jobID, models.JobStatusCompleted, nil)
		return nil
	}

	log.Infow("processor: files discovered", "count", len(paths))
	_ = p.repo.UpdateJobProgress(ctx, jobID, len(paths), 0, 0)

	// ── Worker pool ────────────────────────────────────────────────────────────
	pathCh := make(chan string, len(paths))
	for _, p := range paths {
		pathCh <- p
	}
	close(pathCh)

	var (
		wg       sync.WaitGroup
		doneAtomic  int64
		errorAtomic int64
	)

	for i := 0; i < p.cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wlog := log.With("worker", workerID)
			for path := range pathCh {
				if ctx.Err() != nil {
					return
				}
				if procErr := p.processFile(ctx, jobID, path); procErr != nil {
					atomic.AddInt64(&errorAtomic, 1)
					wlog.Errorw("processor: file failed", "path", path, "err", procErr)
				} else {
					atomic.AddInt64(&doneAtomic, 1)
				}
				// Update progress periodically (every file is fine for small sets)
				_ = p.repo.UpdateJobProgress(ctx, jobID,
					len(paths),
					int(atomic.LoadInt64(&doneAtomic)),
					int(atomic.LoadInt64(&errorAtomic)),
				)
			}
		}(i)
	}

	wg.Wait()

	// ── Finish ─────────────────────────────────────────────────────────────────
	finalStatus := models.JobStatusCompleted
	if atomic.LoadInt64(&errorAtomic) > 0 {
		finalStatus = models.JobStatusFailed
	}

	if err := p.repo.FinishJob(ctx, jobID, finalStatus, nil); err != nil {
		log.Errorw("processor: finish job", "err", err)
	}

	log.Infow("processor: done",
		"total", len(paths),
		"done", atomic.LoadInt64(&doneAtomic),
		"errors", atomic.LoadInt64(&errorAtomic),
		"job_id", jobID,
	)
	return nil
}

// processFile handles one FITS file end-to-end.
func (p *Processor) processFile(ctx context.Context, jobID int64, path string) error {
	log := logger.S().With("path", path)

	// ── Parse ──────────────────────────────────────────────────────────────────
	result, err := ParseFile(path)
	if err != nil {
		_ = p.repo.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FilePath: path,
			Stage:    "parse",
			Message:  err.Error(),
		})
		return fmt.Errorf("parse: %w", err)
	}

	// ── Skip duplicates ────────────────────────────────────────────────────────
	exists, err := p.repo.FileExistsByChecksum(ctx, result.File.Checksum)
	if err != nil {
		log.Warnw("processor: checksum check failed, processing anyway", "err", err)
	}
	if exists {
		log.Infow("processor: skipping unchanged file", "checksum", result.File.Checksum)
		return nil
	}

	// ── Persist file row ───────────────────────────────────────────────────────
	fileID, err := p.repo.UpsertFile(ctx, &result.File)
	if err != nil {
		_ = p.repo.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FilePath: path,
			Stage:    "insert",
			Message:  err.Error(),
		})
		return fmt.Errorf("upsert file: %w", err)
	}

	// ── Delete old headers (in case of re-processing) ─────────────────────────
	if err := p.repo.DeleteHeadersByFileID(ctx, fileID); err != nil {
		log.Warnw("processor: could not delete old headers", "err", err)
	}

	// ── Persist headers ────────────────────────────────────────────────────────
	if err := p.repo.BulkInsertHeaders(ctx, fileID, result.Headers); err != nil {
		_ = p.repo.UpdateFileStatus(ctx, fileID, models.FileStatusError, strPtr(err.Error()))
		_ = p.repo.InsertError(ctx, &models.ProcessingError{
			JobID:    jobID,
			FileID:   &fileID,
			FilePath: path,
			Stage:    "insert",
			Message:  err.Error(),
		})
		return fmt.Errorf("insert headers: %w", err)
	}

	// ── Persist typed metadata ─────────────────────────────────────────────────
	result.Metadata.FileID = fileID
	if err := p.repo.UpsertMetadata(ctx, fileID, &result.Metadata); err != nil {
		log.Warnw("processor: metadata upsert failed", "err", err)
		// Non-fatal — headers are the source of truth
	}

	// ── Mark done ─────────────────────────────────────────────────────────────
	if err := p.repo.UpdateFileStatus(ctx, fileID, models.FileStatusDone, nil); err != nil {
		log.Warnw("processor: could not mark file done", "err", err)
	}

	log.Infow("processor: file processed",
		"file_id", fileID,
		"headers", len(result.Headers),
	)
	return nil
}

func strPtr(s string) *string { return &s }
