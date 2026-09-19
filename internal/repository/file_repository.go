package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/models"
)

// FileRepository handles fits_files persistence.
type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

// UpsertFile inserts or updates a fits_files row by file_path. Returns the row ID.
func (r *FileRepository) UpsertFile(ctx context.Context, tx pgx.Tx, f *models.FITSFile) (int64, error) {
	const q = `
		INSERT INTO fits_files (file_path, file_name, file_size, checksum, hdu_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (file_path) DO UPDATE
			SET file_size  = EXCLUDED.file_size,
			    checksum   = EXCLUDED.checksum,
			    hdu_count  = EXCLUDED.hdu_count,
			    status     = EXCLUDED.status,
			    updated_at = NOW()
		RETURNING id`

	var id int64
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, q, f.FilePath, f.FileName, f.FileSize, f.Checksum, f.HDUCount, string(f.Status)).Scan(&id)
	} else {
		err = r.pool.QueryRow(ctx, q, f.FilePath, f.FileName, f.FileSize, f.Checksum, f.HDUCount, string(f.Status)).Scan(&id)
	}
	if err != nil {
		return 0, fmt.Errorf("file_repo: upsert: %w", err)
	}
	return id, nil
}

// UpdateStatus sets status, error_message, processed_at, and optional processing_ms.
func (r *FileRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id int64, status models.FileStatus, errMsg *string, processingMs *int64) error {
	const q = `
		UPDATE fits_files
		SET status=$2, error_message=$3, processed_at=NOW(), processing_ms=$4, updated_at=NOW()
		WHERE id=$1`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, q, id, string(status), errMsg, processingMs)
	} else {
		_, err = r.pool.Exec(ctx, q, id, string(status), errMsg, processingMs)
	}
	if err != nil {
		return fmt.Errorf("file_repo: update status: %w", err)
	}
	return nil
}

// GetByID returns a single fits_files row.
func (r *FileRepository) GetByID(ctx context.Context, id int64) (*models.FITSFile, error) {
	const q = `
		SELECT id, file_path, file_name, file_size, checksum, hdu_count,
		       status, error_message, skipped_reason, processed_at, created_at, updated_at
		FROM fits_files WHERE id = $1`

	f := &models.FITSFile{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&f.ID, &f.FilePath, &f.FileName, &f.FileSize, &f.Checksum, &f.HDUCount,
		&f.Status, &f.ErrorMsg, &f.SkippedReason, &f.ProcessedAt, &f.CreatedAt, &f.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("file_repo: get by id: %w", err)
	}
	return f, nil
}

// PathDoneWithChecksum returns true if this exact path already has a completed
// row with this checksum — i.e. an idempotent re-scan of an unchanged file.
// This is NOT a "duplicate" (same content under a different name); it's the
// same file seen again and needs no work at all.
func (r *FileRepository) PathDoneWithChecksum(ctx context.Context, path, checksum string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fits_files WHERE file_path=$1 AND checksum=$2 AND status='done')`,
		path, checksum,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("file_repo: path checksum check: %w", err)
	}
	return exists, nil
}

// FindDoneByChecksum returns the first completed file whose content matches
// checksum, excluding excludePath itself. A non-nil result means the file
// currently being scanned is duplicate content of an already-stored file.
func (r *FileRepository) FindDoneByChecksum(ctx context.Context, checksum, excludePath string) (*models.FITSFile, error) {
	const q = `
		SELECT id, file_path, file_name
		FROM fits_files
		WHERE checksum=$1 AND status='done' AND file_path <> $2
		ORDER BY created_at ASC
		LIMIT 1`
	f := &models.FITSFile{}
	err := r.pool.QueryRow(ctx, q, checksum, excludePath).Scan(&f.ID, &f.FilePath, &f.FileName)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("file_repo: find by checksum: %w", err)
	}
	return f, nil
}

// MarkSkipped marks a file row as skipped (e.g. duplicate content) with a
// human-readable reason, without touching header/metadata tables.
func (r *FileRepository) MarkSkipped(ctx context.Context, id int64, reason string) error {
	const q = `
		UPDATE fits_files
		SET status='skipped', skipped_reason=$2, processed_at=NOW(), updated_at=NOW()
		WHERE id=$1`
	if _, err := r.pool.Exec(ctx, q, id, reason); err != nil {
		return fmt.Errorf("file_repo: mark skipped: %w", err)
	}
	return nil
}

// Delete removes a fits_files row (cascades to headers, metadata, overrides).
func (r *FileRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fits_files WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("file_repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── List with filters ─────────────────────────────────────────────────────────

// ListFilesFilter controls filtering and pagination for ListFiles.
type ListFilesFilter struct {
	Search      string           // partial match on file_name
	Status      models.FileStatus
	DateFrom    *time.Time
	DateTo      *time.Time
	SortBy      string           // created_at | file_name | file_size | processed_at
	SortOrder   string           // asc | desc
	Page        int
	PageSize    int
}

type ListFilesResult struct {
	Files []models.FITSFile
	Total int
}

var allowedFileSorts = map[string]string{
	"created_at":   "f.created_at",
	"file_name":    "f.file_name",
	"file_size":    "f.file_size",
	"processed_at": "f.processed_at",
}

func (r *FileRepository) ListFiles(ctx context.Context, f ListFilesFilter) (*ListFilesResult, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf("f.file_name ILIKE $%d", idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("f.status = $%d", idx))
		args = append(args, string(f.Status))
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("f.created_at >= $%d", idx))
		args = append(args, f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("f.created_at <= $%d", idx))
		args = append(args, f.DateTo)
		idx++
	}

	clause := strings.Join(where, " AND ")

	// Count
	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM fits_files f WHERE %s", clause),
		args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("file_repo: count: %w", err)
	}

	// Sort
	sortCol := "f.created_at"
	if col, ok := allowedFileSorts[f.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.ToUpper(f.SortOrder) == "ASC" {
		sortDir = "ASC"
	}

	// Page
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	listArgs := append(args, f.PageSize, offset)
	listQ := fmt.Sprintf(`
		SELECT f.id, f.file_path, f.file_name, f.file_size, f.checksum, f.hdu_count,
		       f.status, f.error_message, f.skipped_reason, f.processed_at, f.created_at, f.updated_at
		FROM fits_files f
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		clause, sortCol, sortDir, idx, idx+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("file_repo: list: %w", err)
	}
	defer rows.Close()

	var files []models.FITSFile
	for rows.Next() {
		var file models.FITSFile
		if err := rows.Scan(
			&file.ID, &file.FilePath, &file.FileName, &file.FileSize,
			&file.Checksum, &file.HDUCount, &file.Status, &file.ErrorMsg, &file.SkippedReason,
			&file.ProcessedAt, &file.CreatedAt, &file.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("file_repo: scan: %w", err)
		}
		files = append(files, file)
	}
	return &ListFilesResult{Files: files, Total: total}, nil
}
