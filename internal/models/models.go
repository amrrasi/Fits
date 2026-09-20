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
	ID            int64      `db:"id" json:"id"`
	FilePath      string     `db:"file_path" json:"file_path"` // absolute path on disk
	FileName      string     `db:"file_name" json:"file_name"` // basename
	FileSize      int64      `db:"file_size" json:"file_size"` // bytes
	Checksum      string     `db:"checksum" json:"checksum"`   // SHA-256 hex
	HDUCount      int        `db:"hdu_count" json:"hdu_count"` // number of HDUs in the file
	Status        FileStatus `db:"status" json:"status"`
	ErrorMsg      *string    `db:"error_message" json:"error_message,omitempty"`   // nullable
	SkippedReason *string    `db:"skipped_reason" json:"skipped_reason,omitempty"` // set when status=skipped (e.g. duplicate content)
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
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
	ID        int64     `db:"id" json:"id"`
	FileID    int64     `db:"file_id" json:"file_id"`
	HDUIndex  int       `db:"hdu_index" json:"hdu_index"` // 0-based
	HDUName   string    `db:"hdu_name" json:"hdu_name"`   // EXTNAME or "PRIMARY"
	Keyword   string    `db:"keyword" json:"keyword"`
	Value     string    `db:"value" json:"value"` // always stored as text
	Comment   string    `db:"comment" json:"comment"`
	ValueType string    `db:"value_type" json:"value_type"` // string | int | float | bool | complex | undefined
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// FITSMetadata stores the well-known typed fields for fast querying.
// Stored in the fits_metadata table.
// ─────────────────────────────────────────────────────────────────────────────

type FITSMetadata struct {
	ID     int64 `db:"id" json:"id"`
	FileID int64 `db:"file_id" json:"file_id"`

	// Image geometry
	NAXIS  *int `db:"naxis" json:"naxis,omitempty"`
	NAXIS1 *int `db:"naxis1" json:"naxis1,omitempty"`
	NAXIS2 *int `db:"naxis2" json:"naxis2,omitempty"`
	NAXIS3 *int `db:"naxis3" json:"naxis3,omitempty"`
	BITPIX *int `db:"bitpix" json:"bitpix,omitempty"`

	// Exposure & timing
	ExpTime   *float64 `db:"exptime" json:"exptime,omitempty"`
	DateObs   *string  `db:"date_obs" json:"date_obs,omitempty"`
	TimeObs   *string  `db:"time_obs" json:"time_obs,omitempty"`
	MJDObs    *float64 `db:"mjd_obs" json:"mjd_obs,omitempty"`
	MJDEnd    *float64 `db:"mjd_end" json:"mjd_end,omitempty"`
	Gain      *float64 `db:"gain" json:"gain,omitempty"`
	ReadNoise *float64 `db:"rdnoise" json:"rdnoise,omitempty"`

	// Camera / instrument
	Instrume *string `db:"instrume" json:"instrume,omitempty"`
	Telescop *string `db:"telescop" json:"telescop,omitempty"`
	Observer *string `db:"observer" json:"observer,omitempty"`
	Object   *string `db:"object" json:"object,omitempty"`
	Origin   *string `db:"origin" json:"origin,omitempty"`

	// Temperature
	SetTemp *float64 `db:"set_temp" json:"set_temp,omitempty"`
	CCDTemp *float64 `db:"ccd_temp" json:"ccd_temp,omitempty"`
	AmbTemp *float64 `db:"amb_temp" json:"amb_temp,omitempty"`

	// Filter
	Filter   *string `db:"filter" json:"filter,omitempty"`
	FilterID *string `db:"filter_id" json:"filter_id,omitempty"`

	// Location
	SiteLatitude  *float64 `db:"site_lat" json:"site_lat,omitempty"`
	SiteLongitude *float64 `db:"site_lon" json:"site_lon,omitempty"`
	SiteElevation *float64 `db:"site_elev" json:"site_elev,omitempty"`

	// Target / astrometry
	RA      *float64 `db:"ra" json:"ra,omitempty"`
	Dec     *float64 `db:"dec" json:"dec,omitempty"`
	Airmass *float64 `db:"airmass" json:"airmass,omitempty"`

	// WCS
	CRVAL1 *float64 `db:"crval1" json:"crval1,omitempty"`
	CRVAL2 *float64 `db:"crval2" json:"crval2,omitempty"`
	CRPIX1 *float64 `db:"crpix1" json:"crpix1,omitempty"`
	CRPIX2 *float64 `db:"crpix2" json:"crpix2,omitempty"`
	CD1_1  *float64 `db:"cd1_1" json:"cd1_1,omitempty"`
	CD1_2  *float64 `db:"cd1_2" json:"cd1_2,omitempty"`
	CD2_1  *float64 `db:"cd2_1" json:"cd2_1,omitempty"`
	CD2_2  *float64 `db:"cd2_2" json:"cd2_2,omitempty"`
	CTYPE1 *string  `db:"ctype1" json:"ctype1,omitempty"`
	CTYPE2 *string  `db:"ctype2" json:"ctype2,omitempty"`

	// Scaling
	BScale *float64 `db:"bscale" json:"bscale,omitempty"`
	BZero  *float64 `db:"bzero" json:"bzero,omitempty"`

	// Extras
	SoftwareVersion *string `db:"software" json:"software,omitempty"`
	EquipID         *string `db:"equip_id" json:"equip_id,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingJob tracks each scan run.
// ─────────────────────────────────────────────────────────────────────────────

type ProcessingJob struct {
	ID             int64      `db:"id" json:"id"`
	ScanDir        string     `db:"scan_dir" json:"scan_dir"`
	Status         JobStatus  `db:"status" json:"status"`
	TotalFiles     int        `db:"total_files" json:"total_files"`
	DoneFiles      int        `db:"done_files" json:"done_files"`
	ErrorFiles     int        `db:"error_files" json:"error_files"`
	DuplicateFiles int        `db:"duplicate_files" json:"duplicate_files"`
	StartedAt      time.Time  `db:"started_at" json:"started_at"`
	FinishedAt     *time.Time `db:"finished_at" json:"finished_at,omitempty"`
	ErrorMsg       *string    `db:"error_message" json:"error_message,omitempty"`
}

type JobStatus string

const (
	JobStatusRunning         JobStatus = "running"
	JobStatusCompleted       JobStatus = "completed"
	JobStatusPartiallyFailed JobStatus = "partially_failed"
	JobStatusFailed          JobStatus = "failed"
	JobStatusCancelled       JobStatus = "cancelled"
)

// ─────────────────────────────────────────────────────────────────────────────
// ProcessingError records a per-file failure inside a job.
// ─────────────────────────────────────────────────────────────────────────────

type ProcessingError struct {
	ID        int64     `db:"id" json:"id"`
	JobID     int64     `db:"job_id" json:"job_id"`
	FileID    *int64    `db:"file_id" json:"file_id,omitempty"` // nullable — may fail before file row exists
	FilePath  string    `db:"file_path" json:"file_path"`
	Stage     string    `db:"stage" json:"stage"` // scan | parse | insert
	Message   string    `db:"message" json:"message"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
