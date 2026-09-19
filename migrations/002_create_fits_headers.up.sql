-- 002_create_fits_headers.up.sql
-- EAV table: every raw keyword from every HDU in every FITS file.
-- This is the complete, lossless record of the original headers.

CREATE TABLE IF NOT EXISTS fits_headers (
    id          BIGSERIAL    PRIMARY KEY,
    file_id     BIGINT       NOT NULL REFERENCES fits_files(id) ON DELETE CASCADE,
    hdu_index   INT          NOT NULL DEFAULT 0,  -- 0 = PRIMARY HDU
    hdu_name    TEXT         NOT NULL DEFAULT 'PRIMARY',
    keyword     TEXT         NOT NULL,
    value       TEXT         NOT NULL DEFAULT '',
    comment     TEXT         NOT NULL DEFAULT '',
    value_type  TEXT         NOT NULL DEFAULT 'string'
                             CHECK (value_type IN ('string','int','float','bool','complex','undefined')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fits_headers_file_id ON fits_headers (file_id);
CREATE INDEX idx_fits_headers_keyword ON fits_headers (keyword);
-- Useful for "find all files where OBJECT = 'M31'"
CREATE INDEX idx_fits_headers_kw_val  ON fits_headers (keyword, value);
