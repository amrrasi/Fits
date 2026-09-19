-- 003_create_fits_metadata.up.sql
-- Typed, flat denormalised table for fast querying of well-known FITS keywords.
-- Sourced from primary HDU of each file. NULL = keyword not present in that file.

CREATE TABLE IF NOT EXISTS fits_metadata (
    id          BIGSERIAL    PRIMARY KEY,
    file_id     BIGINT       NOT NULL UNIQUE REFERENCES fits_files(id) ON DELETE CASCADE,

    -- Image geometry
    naxis       INT,
    naxis1      INT,
    naxis2      INT,
    naxis3      INT,
    bitpix      INT,

    -- Exposure & timing
    exptime     DOUBLE PRECISION,
    date_obs    TEXT,
    time_obs    TEXT,
    mjd_obs     DOUBLE PRECISION,
    mjd_end     DOUBLE PRECISION,
    gain        DOUBLE PRECISION,
    rdnoise     DOUBLE PRECISION,

    -- Camera / instrument
    instrume    TEXT,
    telescop    TEXT,
    observer    TEXT,
    object      TEXT,
    origin      TEXT,
    software    TEXT,
    equip_id    TEXT,

    -- Temperature (°C)
    set_temp    DOUBLE PRECISION,
    ccd_temp    DOUBLE PRECISION,
    amb_temp    DOUBLE PRECISION,

    -- Filter
    filter      TEXT,
    filter_id   TEXT,

    -- Site location
    site_lat    DOUBLE PRECISION,
    site_lon    DOUBLE PRECISION,
    site_elev   DOUBLE PRECISION,

    -- Target / astrometry
    ra          DOUBLE PRECISION,
    dec         DOUBLE PRECISION,
    airmass     DOUBLE PRECISION,

    -- WCS
    crval1      DOUBLE PRECISION,
    crval2      DOUBLE PRECISION,
    crpix1      DOUBLE PRECISION,
    crpix2      DOUBLE PRECISION,
    cd1_1       DOUBLE PRECISION,
    cd1_2       DOUBLE PRECISION,
    cd2_1       DOUBLE PRECISION,
    cd2_2       DOUBLE PRECISION,
    ctype1      TEXT,
    ctype2      TEXT,

    -- Scaling
    bscale      DOUBLE PRECISION,
    bzero       DOUBLE PRECISION,

    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Useful query indexes
CREATE INDEX idx_fits_metadata_object   ON fits_metadata (object);
CREATE INDEX idx_fits_metadata_filter   ON fits_metadata (filter);
CREATE INDEX idx_fits_metadata_date_obs ON fits_metadata (date_obs);
CREATE INDEX idx_fits_metadata_instrume ON fits_metadata (instrume);
