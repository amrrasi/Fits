// Package models defines the core domain structs used throughout the FITS processor.
// These mirror the PostgreSQL schema defined in the migrations/ directory.
package models

import (
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// FITSFile represents a single .fits / .fit file that has been discovered.
// Stored in the fits_files table.
// ─────────────────────────────────────────────────────────────────────────────

type FITSFile struct {
	ID          int64     `db:"id"`
	FilePath    string    `db:"file_path"`    // absolute path on disk
	FileName    string    `db:"file_name"`    // basename
	FileSize    int64     `db:"file_size"`    // bytes
	Checksum    string    `db:"checksum"`     // SHA-256 hex
	HDUCount    int       `db:"hdu_count"`    // number of HDUs in the file
	Status      FileStatus `db:"status"`
	ErrorMsg    *string   `db:"error_message"` // nullable
	ProcessedAt *time.Time `db:"processed_at"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// FileStatus is the processing state of a FITS file.
type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"
	FileStatusProcessing FileStatus = "processing"
	FileStatusDone       FileStatus = "done"
	FileStatusError      FileStatus = "error"
	FileStatusSkipped    FileStatus = "skipped" // duplicate / unchanged
)

// ─────────────────────────────────────────────────────────────────────────────
// FITSHeader stores every raw keyword from every HDU (EAV pattern).
// Stored in the fits_headers table.
// ─────────────────────────────────────────────────────────────────────────────

type FITSHeader struct {
	ID         int64     `db:"id"`
	FileID     int64     `db:"file_id"`
	HDUIndex   int       `db:"hdu_index"`   // 0-based
	HDUName    string    `db:"hdu_name"`    // EXTNAME or "PRIMARY"
	Keyword    string    `db:"keyword"`
	Value      string    `db:"value"`       // always stored as text
	Comment    string    `db:"comment"`
	ValueType  string    `db:"value_type"`  // string | int | float | bool | complex | undefined
	CreatedAt  time.Time `db:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// FITSMetadata stores the well-known typed fields for fast querying.
// Stored in the fits_metadata table.
// ─────────────────────────────────────────────────────────────────────────────

type FITSMetadata struct {
	ID     int64 `db:"id"`
	FileID int64 `db:"file_id"`

	// Image geometry
	NAXIS  *int    `db:"naxis"`
	NAXIS1 *int    `db:"naxis1"`
	NAXIS2 *int    `db:"naxis2"`
	NAXIS3 *int    `db:"naxis3"`
	BITPIX *int    `db:"bitpix"`

	// Exposure & timing
	ExpTime   *float64   `db:"exptime"`
	DateObs   *string    `db:"date_obs"`
	TimeObs   *string    `db:"time_obs"`
	MJDObs    *float64   `db:"mjd_obs"`
	MJDEnd    *float64   `db:"mjd_end"`
	Gain      *float64   `db:"gain"`
	ReadNoise *float64   `db:"rdnoise"`

	// Camera / instrument
	Instrume *string `db:"instrume"`
	Telescop *string `db:"telescop"`
	Observer *string `db:"observer"`
	Object   *string `db:"object"`
	Origin   *string `db:"origin"`

	// Temperature
	SetTemp  *float64 `db:"set_temp"`
	CCDTemp  *float64 `db:"ccd_temp"`
	AmbTemp  *float64 `db:"amb_temp"`

	// Filter
	Filter   *string `db:"filter"`
	FilterID *string `db:"filter_id"`

	// Location
	SiteLatitude  *float64 `db:"site_lat"`
	SiteLongitude *float64 `db:"site_lon"`
	SiteElevation *float64 `db:"site_elev"`

	// Target / astrometry
	RA      *float64 `db:"ra"`
	Dec     *float64 `db:"dec"`
	Airmass *float64 `db:"airmass"`

	// WCS
	CRVAL1 *float64 `db:"crval1"`
	CRVAL2 *float64 `db:"crval2"`
	CRPIX1 *float64 `db:"crpix1"`
	CRPIX2 *float64 `db:"crpix2"`
	CD1_1  *float64 `db:"cd1_1"`
	CD1_2  *float64 `db:"cd1_2"`
	CD2_1  *float64 `db:"cd2_1"`
	CD2_2  *float64 `db:"cd2_2"`
	CTYPE1 *string  `db:"ctype1"`
	CTYPE2 *string  `db:"ctype2"`

	// Scaling
	BScale *float64 `db:"bscale"`
	BZero  *float64 `db:"bzero"`

	// Extras
	SoftwareVersion *string `db:"software"`
	EquipID         *string `db:"equip_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingJob tracks each scan run.
// ─────────────────────────────────────────────────────────────────────────────

type ProcessingJob struct {
	ID          int64      `db:"id"`
	ScanDir     string     `db:"scan_dir"`
	Status      JobStatus  `db:"status"`
	TotalFiles  int        `db:"total_files"`
	DoneFiles   int        `db:"done_files"`
	ErrorFiles  int        `db:"error_files"`
	StartedAt   time.Time  `db:"started_at"`
	FinishedAt  *time.Time `db:"finished_at"`
	ErrorMsg    *string    `db:"error_message"`
}

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingError records a per-file failure inside a job.
// ─────────────────────────────────────────────────────────────────────────────

type ProcessingError struct {
	ID        int64     `db:"id"`
	JobID     int64     `db:"job_id"`
	FileID    *int64    `db:"file_id"` // nullable — may fail before file row exists
	FilePath  string    `db:"file_path"`
	Stage     string    `db:"stage"` // scan | parse | insert
	Message   string    `db:"message"`
	CreatedAt time.Time `db:"created_at"`
}
