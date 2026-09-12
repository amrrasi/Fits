// Package repository contains all database access logic for the FITS processor.
// Each method opens a transaction, does its work, and commits (or rolls back on error).
// The rest of the application never writes SQL directly — it calls this package.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
)

// FITSRepository handles all persistence for FITS files, headers, and metadata.
type FITSRepository struct {
	pool *pgxpool.Pool
}

// NewFITSRepository creates a new repository backed by the given pool.
func NewFITSRepository(pool *pgxpool.Pool) *FITSRepository {
	return &FITSRepository{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// FITSFile
// ─────────────────────────────────────────────────────────────────────────────

// UpsertFile inserts a fits_files row or, if the same file path already exists,
// updates the checksum and status.  Returns the row's ID.
func (r *FITSRepository) UpsertFile(ctx context.Context, f *models.FITSFile) (int64, error) {
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
	err := r.pool.QueryRow(ctx, q,
		f.FilePath,
		f.FileName,
		f.FileSize,
		f.Checksum,
		f.HDUCount,
		string(f.Status),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("repository: upsert fits_file: %w", err)
	}
	return id, nil
}

// UpdateFileStatus sets the status (and optional error message) of a FITS file.
func (r *FITSRepository) UpdateFileStatus(ctx context.Context, id int64, status models.FileStatus, errMsg *string) error {
	const q = `
		UPDATE fits_files
		SET status = $2, error_message = $3, processed_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, q, id, string(status), errMsg)
	if err != nil {
		return fmt.Errorf("repository: update file status: %w", err)
	}
	return nil
}

// FileExistsByChecksum returns true if a file with that checksum is already processed.
// Use this to skip re-processing unchanged files.
func (r *FITSRepository) FileExistsByChecksum(ctx context.Context, checksum string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM fits_files WHERE checksum = $1 AND status = 'done')`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, checksum).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository: check checksum: %w", err)
	}
	return exists, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FITSHeader (EAV table) — bulk copy for performance
// ─────────────────────────────────────────────────────────────────────────────

// BulkInsertHeaders inserts header rows using pgx's CopyFrom (very fast).
// fileID is set on every row before insertion.
func (r *FITSRepository) BulkInsertHeaders(ctx context.Context, fileID int64, headers []models.FITSHeader) error {
	if len(headers) == 0 {
		return nil
	}

	log := logger.S().With("file_id", fileID, "count", len(headers))
	log.Debug("repository: bulk inserting headers")

	rows := make([][]interface{}, 0, len(headers))
	now := time.Now()
	for _, h := range headers {
		rows = append(rows, []interface{}{
			fileID,
			h.HDUIndex,
			h.HDUName,
			h.Keyword,
			h.Value,
			h.Comment,
			h.ValueType,
			now,
		})
	}

	cols := []string{"file_id", "hdu_index", "hdu_name", "keyword", "value", "comment", "value_type", "created_at"}

	n, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"fits_headers"},
		cols,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("repository: bulk insert headers: %w", err)
	}

	log.Infow("repository: headers inserted", "rows", n)
	return nil
}

// DeleteHeadersByFileID removes all header rows for a file (used on re-processing).
func (r *FITSRepository) DeleteHeadersByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM fits_headers WHERE file_id = $1`, fileID)
	if err != nil {
		return fmt.Errorf("repository: delete headers: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FITSMetadata (typed flat table)
// ─────────────────────────────────────────────────────────────────────────────

// UpsertMetadata inserts or replaces the typed metadata row for a file.
func (r *FITSRepository) UpsertMetadata(ctx context.Context, fileID int64, m *models.FITSMetadata) error {
	const q = `
		INSERT INTO fits_metadata (
			file_id,
			naxis, naxis1, naxis2, naxis3, bitpix,
			exptime, date_obs, time_obs, mjd_obs, mjd_end, gain, rdnoise,
			instrume, telescop, observer, object, origin, software, equip_id,
			set_temp, ccd_temp, amb_temp,
			filter, filter_id,
			site_lat, site_lon, site_elev,
			ra, dec, airmass,
			crval1, crval2, crpix1, crpix2, cd1_1, cd1_2, cd2_1, cd2_2, ctype1, ctype2,
			bscale, bzero,
			created_at, updated_at
		) VALUES (
			$1,
			$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,$12,$13,
			$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,
			$24,$25,
			$26,$27,$28,
			$29,$30,$31,
			$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,
			$42,$43,
			NOW(), NOW()
		)
		ON CONFLICT (file_id) DO UPDATE SET
			naxis=$2, naxis1=$3, naxis2=$4, naxis3=$5, bitpix=$6,
			exptime=$7, date_obs=$8, time_obs=$9, mjd_obs=$10, mjd_end=$11, gain=$12, rdnoise=$13,
			instrume=$14, telescop=$15, observer=$16, object=$17, origin=$18, software=$19, equip_id=$20,
			set_temp=$21, ccd_temp=$22, amb_temp=$23,
			filter=$24, filter_id=$25,
			site_lat=$26, site_lon=$27, site_elev=$28,
			ra=$29, dec=$30, airmass=$31,
			crval1=$32, crval2=$33, crpix1=$34, crpix2=$35,
			cd1_1=$36, cd1_2=$37, cd2_1=$38, cd2_2=$39, ctype1=$40, ctype2=$41,
			bscale=$42, bzero=$43,
			updated_at=NOW()`

	_, err := r.pool.Exec(ctx, q,
		fileID,
		m.NAXIS, m.NAXIS1, m.NAXIS2, m.NAXIS3, m.BITPIX,
		m.ExpTime, m.DateObs, m.TimeObs, m.MJDObs, m.MJDEnd, m.Gain, m.ReadNoise,
		m.Instrume, m.Telescop, m.Observer, m.Object, m.Origin, m.SoftwareVersion, m.EquipID,
		m.SetTemp, m.CCDTemp, m.AmbTemp,
		m.Filter, m.FilterID,
		m.SiteLatitude, m.SiteLongitude, m.SiteElevation,
		m.RA, m.Dec, m.Airmass,
		m.CRVAL1, m.CRVAL2, m.CRPIX1, m.CRPIX2,
		m.CD1_1, m.CD1_2, m.CD2_1, m.CD2_2, m.CTYPE1, m.CTYPE2,
		m.BScale, m.BZero,
	)
	if err != nil {
		return fmt.Errorf("repository: upsert metadata: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingJob
// ─────────────────────────────────────────────────────────────────────────────

// CreateJob inserts a new processing_jobs row and returns its ID.
func (r *FITSRepository) CreateJob(ctx context.Context, scanDir string) (int64, error) {
	const q = `
		INSERT INTO processing_jobs (scan_dir, status, total_files, done_files, error_files, started_at)
		VALUES ($1, 'running', 0, 0, 0, NOW())
		RETURNING id`

	var id int64
	if err := r.pool.QueryRow(ctx, q, scanDir).Scan(&id); err != nil {
		return 0, fmt.Errorf("repository: create job: %w", err)
	}
	return id, nil
}

// UpdateJobProgress updates counters on a running job.
func (r *FITSRepository) UpdateJobProgress(ctx context.Context, jobID int64, total, done, errCount int) error {
	const q = `
		UPDATE processing_jobs
		SET total_files=$2, done_files=$3, error_files=$4
		WHERE id=$1`
	_, err := r.pool.Exec(ctx, q, jobID, total, done, errCount)
	if err != nil {
		return fmt.Errorf("repository: update job progress: %w", err)
	}
	return nil
}

// FinishJob marks a job as completed or failed.
func (r *FITSRepository) FinishJob(ctx context.Context, jobID int64, status models.JobStatus, errMsg *string) error {
	const q = `
		UPDATE processing_jobs
		SET status=$2, finished_at=NOW(), error_message=$3
		WHERE id=$1`
	_, err := r.pool.Exec(ctx, q, jobID, string(status), errMsg)
	if err != nil {
		return fmt.Errorf("repository: finish job: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingError
// ─────────────────────────────────────────────────────────────────────────────

// InsertError records a per-file processing failure.
func (r *FITSRepository) InsertError(ctx context.Context, e *models.ProcessingError) error {
	const q = `
		INSERT INTO processing_errors (job_id, file_id, file_path, stage, message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`
	_, err := r.pool.Exec(ctx, q, e.JobID, e.FileID, e.FilePath, e.Stage, e.Message)
	if err != nil {
		return fmt.Errorf("repository: insert error: %w", err)
	}
	return nil
}
