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

// HeaderRepository handles fits_headers persistence.
type HeaderRepository struct {
	pool *pgxpool.Pool
}

func NewHeaderRepository(pool *pgxpool.Pool) *HeaderRepository {
	return &HeaderRepository{pool: pool}
}

// BulkInsert inserts all headers for a file using pgx CopyFrom (fastest method).
// Must be called inside a transaction — pass the tx.
func (r *HeaderRepository) BulkInsert(ctx context.Context, tx pgx.Tx, fileID int64, headers []models.FITSHeader) error {
	if len(headers) == 0 {
		return nil
	}

	now := time.Now()
	rows := make([][]interface{}, 0, len(headers))
	for _, h := range headers {
		rows = append(rows, []interface{}{
			fileID, h.HDUIndex, h.HDUName,
			h.Keyword, h.Value, h.Comment, h.ValueType, now,
		})
	}

	cols := []string{"file_id", "hdu_index", "hdu_name", "keyword", "value", "comment", "value_type", "created_at"}

	_, err := tx.CopyFrom(ctx, pgx.Identifier{"fits_headers"}, cols, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("header_repo: bulk insert: %w", err)
	}
	return nil
}

// DeleteByFileID removes all headers for a file (called before re-insert on reprocessing).
func (r *HeaderRepository) DeleteByFileID(ctx context.Context, tx pgx.Tx, fileID int64) error {
	_, err := tx.Exec(ctx, `DELETE FROM fits_headers WHERE file_id=$1`, fileID)
	if err != nil {
		return fmt.Errorf("header_repo: delete: %w", err)
	}
	return nil
}

// ListHeadersFilter paginates and filters headers for one file.
type ListHeadersFilter struct {
	FileID   int64
	HDUIndex *int    // nil = all HDUs
	Search   string  // partial match on keyword or value
	Page     int
	PageSize int
}

type ListHeadersResult struct {
	Headers []models.FITSHeader
	Total   int
}

func (r *HeaderRepository) ListHeaders(ctx context.Context, f ListHeadersFilter) (*ListHeadersResult, error) {
	where := []string{"file_id=$1"}
	args := []interface{}{f.FileID}
	idx := 2

	if f.HDUIndex != nil {
		where = append(where, fmt.Sprintf("hdu_index=$%d", idx))
		args = append(args, *f.HDUIndex)
		idx++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("(keyword ILIKE $%d OR value ILIKE $%d)", idx, idx+1))
		pat := "%" + f.Search + "%"
		args = append(args, pat, pat)
		idx += 2
	}

	clause := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM fits_headers WHERE %s", clause),
		args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("header_repo: count: %w", err)
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 100
	}
	offset := (f.Page - 1) * f.PageSize

	listArgs := append(args, f.PageSize, offset)
	listQ := fmt.Sprintf(`
		SELECT id, file_id, hdu_index, hdu_name, keyword, value, comment, value_type, created_at
		FROM fits_headers
		WHERE %s
		ORDER BY hdu_index ASC, id ASC
		LIMIT $%d OFFSET $%d`, clause, idx, idx+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("header_repo: list: %w", err)
	}
	defer rows.Close()

	var headers []models.FITSHeader
	for rows.Next() {
		var h models.FITSHeader
		if err := rows.Scan(
			&h.ID, &h.FileID, &h.HDUIndex, &h.HDUName,
			&h.Keyword, &h.Value, &h.Comment, &h.ValueType, &h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("header_repo: scan: %w", err)
		}
		headers = append(headers, h)
	}
	return &ListHeadersResult{Headers: headers, Total: total}, nil
}
