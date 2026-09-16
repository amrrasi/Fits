-- 007_duplicate_tracking.up.sql
-- Adds a per-job counter for files skipped because their content (checksum)
-- already exists under a different path. The per-file reason is stored in
-- fits_files.skipped_reason, which was added in 006 but never populated.

ALTER TABLE processing_jobs
    ADD COLUMN IF NOT EXISTS duplicate_files INT NOT NULL DEFAULT 0;
