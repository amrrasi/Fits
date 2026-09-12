-- 001_create_fits_files.up.sql
-- Stores one row per FITS file discovered on disk.

CREATE TABLE IF NOT EXISTS fits_files (
    id            BIGSERIAL    PRIMARY KEY,
    file_path     TEXT         NOT NULL UNIQUE,   -- absolute path on disk
    file_name     TEXT         NOT NULL,
    file_size     BIGINT       NOT NULL DEFAULT 0,
    checksum      TEXT         NOT NULL,           -- SHA-256 hex
    hdu_count     INT          NOT NULL DEFAULT 0,
    status        TEXT         NOT NULL DEFAULT 'pending'
                               CHECK (status IN ('pending','processing','done','error','skipped')),
    error_message TEXT,
    processed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fits_files_status   ON fits_files (status);
CREATE INDEX idx_fits_files_checksum ON fits_files (checksum);
CREATE INDEX idx_fits_files_filename ON fits_files (file_name);
