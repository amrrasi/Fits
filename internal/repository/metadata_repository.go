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

// MetadataRepository handles fits_metadata and metadata_overrides.
type MetadataRepository struct {
	pool *pgxpool.Pool
}

func NewMetadataRepository(pool *pgxpool.Pool) *MetadataRepository {
	return &MetadataRepository{pool: pool}
}

// Upsert inserts or replaces the typed metadata row. Must be called in a transaction.
func (r *MetadataRepository) Upsert(ctx context.Context, tx pgx.Tx, fileID int64, m *models.FITSMetadata) error {
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
			NOW(),NOW()
		)
		ON CONFLICT (file_id) DO UPDATE SET
			naxis=$2,naxis1=$3,naxis2=$4,naxis3=$5,bitpix=$6,
			exptime=$7,date_obs=$8,time_obs=$9,mjd_obs=$10,mjd_end=$11,gain=$12,rdnoise=$13,
			instrume=$14,telescop=$15,observer=$16,object=$17,origin=$18,software=$19,equip_id=$20,
			set_temp=$21,ccd_temp=$22,amb_temp=$23,
			filter=$24,filter_id=$25,
			site_lat=$26,site_lon=$27,site_elev=$28,
			ra=$29,dec=$30,airmass=$31,
			crval1=$32,crval2=$33,crpix1=$34,crpix2=$35,
			cd1_1=$36,cd1_2=$37,cd2_1=$38,cd2_2=$39,ctype1=$40,ctype2=$41,
			bscale=$42,bzero=$43,
			updated_at=NOW()`

	exec := func(q string, args ...interface{}) error {
		if tx != nil {
			_, err := tx.Exec(ctx, q, args...)
			return err
		}
		_, err := r.pool.Exec(ctx, q, args...)
		return err
	}

	err := exec(q,
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
		return fmt.Errorf("metadata_repo: upsert: %w", err)
	}
	return nil
}

// GetByFileID returns the typed metadata for a file.
func (r *MetadataRepository) GetByFileID(ctx context.Context, fileID int64) (*models.FITSMetadata, error) {
	const q = `
		SELECT id,file_id,
			naxis,naxis1,naxis2,naxis3,bitpix,
			exptime,date_obs,time_obs,mjd_obs,mjd_end,gain,rdnoise,
			instrume,telescop,observer,object,origin,software,equip_id,
			set_temp,ccd_temp,amb_temp,
			filter,filter_id,
			site_lat,site_lon,site_elev,
			ra,dec,airmass,
			crval1,crval2,crpix1,crpix2,cd1_1,cd1_2,cd2_1,cd2_2,ctype1,ctype2,
			bscale,bzero,
			created_at,updated_at
		FROM fits_metadata WHERE file_id=$1`

	m := &models.FITSMetadata{}
	err := r.pool.QueryRow(ctx, q, fileID).Scan(
		&m.ID, &m.FileID,
		&m.NAXIS, &m.NAXIS1, &m.NAXIS2, &m.NAXIS3, &m.BITPIX,
		&m.ExpTime, &m.DateObs, &m.TimeObs, &m.MJDObs, &m.MJDEnd, &m.Gain, &m.ReadNoise,
		&m.Instrume, &m.Telescop, &m.Observer, &m.Object, &m.Origin, &m.SoftwareVersion, &m.EquipID,
		&m.SetTemp, &m.CCDTemp, &m.AmbTemp,
		&m.Filter, &m.FilterID,
		&m.SiteLatitude, &m.SiteLongitude, &m.SiteElevation,
		&m.RA, &m.Dec, &m.Airmass,
		&m.CRVAL1, &m.CRVAL2, &m.CRPIX1, &m.CRPIX2,
		&m.CD1_1, &m.CD1_2, &m.CD2_1, &m.CD2_2, &m.CTYPE1, &m.CTYPE2,
		&m.BScale, &m.BZero,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("metadata_repo: get: %w", err)
	}
	return m, nil
}

// ListMetadataFilter for the files list with science filters.
type ListMetadataFilter struct {
	Object      string
	Filter      string
	Instrume    string
	Telescop    string
	DateFrom    *time.Time
	DateTo      *time.Time
	ExpTimeMin  *float64
	ExpTimeMax  *float64
	RAMin       *float64
	RAMax       *float64
	DecMin      *float64
	DecMax      *float64
}

// ── Overrides ──────────────────────────────────────────────────────────────────

// MetadataOverride is one edit record from metadata_overrides.
type MetadataOverride struct {
	ID            int64      `json:"id"`
	FileID        int64      `json:"file_id"`
	FieldName     string     `json:"field_name"`
	OriginalValue *string    `json:"original_value"`
	NewValue      string     `json:"new_value"`
	Reason        *string    `json:"reason"`
	EditedBy      *int64     `json:"edited_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

// InsertOverride records a metadata edit. Does NOT apply the change to fits_metadata —
// the caller (service) must also call a targeted UPDATE on fits_metadata.
func (r *MetadataRepository) InsertOverride(ctx context.Context, o *MetadataOverride) error {
	const q = `
		INSERT INTO metadata_overrides (file_id, field_name, original_value, new_value, reason, edited_by)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, q, o.FileID, o.FieldName, o.OriginalValue, o.NewValue, o.Reason, o.EditedBy)
	if err != nil {
		return fmt.Errorf("metadata_repo: insert override: %w", err)
	}
	return nil
}

// ListOverrides returns all override history for a file.
func (r *MetadataRepository) ListOverrides(ctx context.Context, fileID int64) ([]MetadataOverride, error) {
	const q = `
		SELECT id, file_id, field_name, original_value, new_value, reason, edited_by, created_at
		FROM metadata_overrides
		WHERE file_id=$1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, fileID)
	if err != nil {
		return nil, fmt.Errorf("metadata_repo: list overrides: %w", err)
	}
	defer rows.Close()

	var result []MetadataOverride
	for rows.Next() {
		var o MetadataOverride
		if err := rows.Scan(&o.ID, &o.FileID, &o.FieldName, &o.OriginalValue, &o.NewValue, &o.Reason, &o.EditedBy, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("metadata_repo: scan override: %w", err)
		}
		result = append(result, o)
	}
	return result, nil
}

// ApplyFieldUpdate writes a single field change to fits_metadata.
// fieldName must be in the allowed list — validated by the service before calling.
func (r *MetadataRepository) ApplyFieldUpdate(ctx context.Context, fileID int64, fieldName, newValue string) error {
	allowed := allowedEditFields()
	if _, ok := allowed[fieldName]; !ok {
		return fmt.Errorf("metadata_repo: field %q is not editable", fieldName)
	}
	q := fmt.Sprintf(
		`UPDATE fits_metadata SET %s=$2, updated_at=NOW() WHERE file_id=$1`,
		strings.ToLower(fieldName),
	)
	tag, err := r.pool.Exec(ctx, q, fileID, newValue)
	if err != nil {
		return fmt.Errorf("metadata_repo: apply field update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// allowedEditFields returns the set of fits_metadata column names that editors may change.
// Add new editable fields here — they are validated at the repository level.
func allowedEditFields() map[string]struct{} {
	fields := []string{
		"object", "observer", "telescop", "instrume", "filter", "filter_id",
		"origin", "software", "equip_id",
		"ra", "dec", "airmass",
		"site_lat", "site_lon", "site_elev",
		"exptime", "gain", "rdnoise",
		"set_temp", "ccd_temp", "amb_temp",
		"date_obs", "time_obs",
	}
	m := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		m[f] = struct{}{}
	}
	return m
}

// AllowedEditFields is exported so the handler can validate before calling.
func AllowedEditFields() map[string]struct{} { return allowedEditFields() }
