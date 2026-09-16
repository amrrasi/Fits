-- 007_duplicate_tracking.down.sql

ALTER TABLE processing_jobs
    DROP COLUMN IF EXISTS duplicate_files;
