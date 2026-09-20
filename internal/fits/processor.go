package fits

import (
	"context"
	"errors"
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

	baseCtx context.Context
	wg      sync.WaitGroup
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

// ErrScanAlreadyRunning is returned when another scan (API or CLI, any process) holds the lock.
var ErrScanAlreadyRunning = errors.New("یک اسکن دیگر هم‌اکنون در حال اجراست")

// scanLockKey is a PostgreSQL advisory-lock key. The lock lives on a dedicated connection, so it
// is released automatically if the process dies - a crash can never leave scanning blocked.
const scanLockKey int64 = 0x46495453

const scanTimeout = 6 * time.Hour

// SetBaseContext sets the context that background scans inherit (cancelled on shutdown).
func (p *Processor) SetBaseContext(ctx context.Context) { p.baseCtx = ctx }

// Wait blocks until running background scans finish or the timeout passes.
func (p *Processor) Wait(timeout time.Duration) {
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (p *Processor) lock(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("پردازشگر: acquire conn: %w", err)
	}
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, scanLockKey).Scan(&ok); err != nil {
		conn.Release()
		return nil, fmt.Errorf("پردازشگر: advisory lock: %w", err)
	}
	if !ok {
		conn.Release()
		return nil, ErrScanAlreadyRunning
	}
	// we hold the lock => no live scan exists => any 'running' row is an orphan from a crash
	if _, err := conn.Exec(ctx, `UPDATE processing_jobs SET status='failed', finished_at=NOW(),
		error_message='اسکن به‌صورت ناگهانی متوقف شده بود' WHERE status='running'`); err != nil {
		logger.S().Warnw("پردازشگر: orphan job cleanup failed", "err", err)
	}
	return conn, nil
}

func (p *Processor) unlock(conn *pgxpool.Conn) {
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := conn.Exec(c, `SELECT pg_advisory_unlock($1)`, scanLockKey); err != nil {
		logger.S().Warnw("پردازشگر: advisory unlock failed", "err", err)
		conn.Conn().Close(c) // dropping the connection releases the lock
	}
	conn.Release()
}

// finish records the final job state using a fresh context (the scan ctx may already be cancelled).
func (p *Processor) finish(jobID int64, status models.JobStatus, durationMs int64, msg *string) {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.jobs.Finish(c, jobID, status, durationMs, msg); err != nil {
		logger.S().Errorw("پردازشگر: finish job failed", "job_id", jobID, "err", err)
	}
}

// Start begins a background scan of scanDir (already validated by the caller) and returns its job id.
func (p *Processor) Start(scanDir string) (int64, error) {
	base := p.baseCtx
	if base == nil {
		base = context.Background()
	}
	conn, err := p.lock(base)
	if err != nil {
		return 0, err
	}
	jobID, err := p.jobs.Create(base, scanDir)
	if err != nil {
		p.unlock(conn)
		return 0, fmt.Errorf("پردازشگر: ساخت جاب: %w", err)
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer p.unlock(conn)
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("panic: %v", r)
				logger.S().Errorw("پردازشگر: panic در اسکن", "job_id", jobID, "panic", r)
				p.finish(jobID, models.JobStatusFailed, 0, &msg)
			}
		}()
		ctx, cancel := context.WithTimeout(base, scanTimeout)
		defer cancel()
		if err := p.execute(ctx, jobID, scanDir); err != nil {
			logger.S().Errorw("پردازشگر: scan failed", "job_id", jobID, "err", err)
		}
	}()
	return jobID, nil
}

// Run scans the configured directory synchronously (CLI mode).
func (p *Processor) Run(ctx context.Context) error {
	conn, err := p.lock(ctx)
	if err != nil {
		return err
	}
	defer p.unlock(conn)
	jobID, err := p.jobs.Create(ctx, p.cfg.ScanDir)
	if err != nil {
		return fmt.Errorf("پردازشگر: ساخت جاب: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()
	return p.execute(ctx, jobID, p.cfg.ScanDir)
}

func (p *Processor) execute(ctx context.Context, jobID int64, scanTarget string) error {
	start := time.Now()
	log := logger.S().With("scan_dir", scanTarget, "job_id", jobID)
	log.Infow("پردازشگر: پردازش شروع شد")

	paths, err := ScanDir(scanTarget)
	if err != nil {
		msg := err.Error()
		p.finish(jobID, models.JobStatusFailed, 0, &msg)
		return fmt.Errorf("پردازشگر: اسکن: %w", err)
	}
	if len(paths) == 0 {
		log.Warn("پردازشگر: فایل فیتسی یافت نشد")
		p.finish(jobID, models.JobStatusCompleted, time.Since(start).Milliseconds(), nil)
		return nil
	}

	log.Infow("پردازشگر: فایل‌های یافت‌شده", "تعداد", len(paths))
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
					wlog.Errorw("پردازشگر: ناموفق", "path", path, "err", procErr)
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

	p.finish(jobID, finalStatus, durationMs, nil)

	log.Infow("پردازشگر: انجام شد ",
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
		// a failed statement aborts the whole transaction, so this cannot be "non-fatal"
		_ = p.jobs.InsertError(ctx, &models.ProcessingError{
			JobID: jobID, FileID: &fileID, FilePath: path, Stage: "insert", Message: err.Error(),
		})
		return "", fmt.Errorf("upsert metadata: %w", err)
	}
	if n, err := p.metadata.ReapplyOverrides(ctx, tx, fileID); err != nil {
		return "", fmt.Errorf("reapply overrides: %w", err)
	} else if n > 0 {
		log.Infow("پردازشگر: ویرایش‌های دستی حفظ شد", "file_id", fileID, "fields", n)
	}

	processingMs := time.Since(start).Milliseconds()
	if err := p.files.UpdateStatus(ctx, tx, fileID, models.FileStatusDone, nil, &processingMs); err != nil {
		return "", fmt.Errorf("update file status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("پردازشگر: commit tx: %w", err)
	}
	tx = nil

	log.Infow("پردازشگر: file done",
		"file_id", fileID,
		"headers", len(result.Headers),
		"processing_ms", processingMs,
	)
	return fileStatusOK, nil
}
