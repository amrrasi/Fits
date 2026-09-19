-- 006_phase4_improvements.up.sql
-- Adds: processing_ms to fits_files, expanded job status enum,
--       metadata_overrides table, audit_logs table,
--       partial/composite indexes for Phase 4 query patterns.

-- ── fits_files improvements ──────────────────────────────────────────────────
ALTER TABLE fits_files
    ADD COLUMN IF NOT EXISTS processing_ms BIGINT,          -- how long ingestion took
    ADD COLUMN IF NOT EXISTS skipped_reason TEXT;           -- why it was skipped

-- ── Expand processing_jobs status ────────────────────────────────────────────
-- Add new status values to the CHECK constraint
ALTER TABLE processing_jobs DROP CONSTRAINT IF EXISTS processing_jobs_status_check;
ALTER TABLE processing_jobs
    ADD CONSTRAINT processing_jobs_status_check
    CHECK (status IN ('running','completed','partially_failed','failed','cancelled'));

ALTER TABLE processing_jobs
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT;

-- ── metadata_overrides ────────────────────────────────────────────────────────
-- Editors/admins can override typed metadata without touching raw headers.
-- Raw fits_headers is always the canonical source; this table stores corrections.
CREATE TABLE IF NOT EXISTS metadata_overrides (
    id            BIGSERIAL    PRIMARY KEY,
    file_id       BIGINT       NOT NULL REFERENCES fits_files(id) ON DELETE CASCADE,
    field_name    TEXT         NOT NULL,   -- column name in fits_metadata (e.g. "ra", "object")
    original_value TEXT,                  -- value at time of override
    new_value     TEXT         NOT NULL,
    reason        TEXT,                   -- optional free-text reason for the change
    edited_by     BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meta_overrides_file_id ON metadata_overrides (file_id);
CREATE INDEX idx_meta_overrides_field   ON metadata_overrides (field_name);
CREATE INDEX idx_meta_overrides_editor  ON metadata_overrides (edited_by);

-- ── audit_logs ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_logs (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       REFERENCES users(id) ON DELETE SET NULL,  -- NULL = system action
    action       TEXT         NOT NULL,   -- e.g. user.create, metadata.edit, file.delete, scan.trigger
    entity_type  TEXT         NOT NULL,   -- e.g. user, fits_file, metadata
    entity_id    TEXT,                    -- string so it works for any PK type
    old_value    JSONB,
    new_value    JSONB,
    ip_address   TEXT,
    request_id   TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user_id     ON audit_logs (user_id);
CREATE INDEX idx_audit_action      ON audit_logs (action);
CREATE INDEX idx_audit_entity      ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_created_at  ON audit_logs (created_at DESC);

-- ── Performance indexes for Phase 4 query patterns ────────────────────────────
-- Files list: filter by status + sort by created_at
CREATE INDEX IF NOT EXISTS idx_fits_files_status_created
    ON fits_files (status, created_at DESC);

-- Metadata: common science filters
CREATE INDEX IF NOT EXISTS idx_fits_metadata_object_filter
    ON fits_metadata (object, filter);

CREATE INDEX IF NOT EXISTS idx_fits_metadata_date_obs
    ON fits_metadata (date_obs);

CREATE INDEX IF NOT EXISTS idx_fits_metadata_instrume
    ON fits_metadata (instrume);

-- Headers: fast lookup per file + keyword
CREATE INDEX IF NOT EXISTS idx_fits_headers_file_keyword
    ON fits_headers (file_id, keyword);

-- Partial index: only files not yet done (used by scanner to skip duplicates)
CREATE INDEX IF NOT EXISTS idx_fits_files_pending
    ON fits_files (file_path) WHERE status != 'done';
