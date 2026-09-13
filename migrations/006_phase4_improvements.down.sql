-- 006_phase4_improvements.down.sql
DROP INDEX IF EXISTS idx_fits_files_pending;
DROP INDEX IF EXISTS idx_fits_metadata_instrume;
DROP INDEX IF EXISTS idx_fits_metadata_date_obs;
DROP INDEX IF EXISTS idx_fits_metadata_object_filter;
DROP INDEX IF EXISTS idx_fits_files_status_created;
DROP INDEX IF EXISTS idx_fits_headers_file_keyword;
DROP INDEX IF EXISTS idx_audit_created_at;
DROP INDEX IF EXISTS idx_audit_entity;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_user_id;
DROP TABLE IF EXISTS audit_logs;
DROP INDEX IF EXISTS idx_meta_overrides_editor;
DROP INDEX IF EXISTS idx_meta_overrides_field;
DROP INDEX IF EXISTS idx_meta_overrides_file_id;
DROP TABLE IF EXISTS metadata_overrides;
ALTER TABLE processing_jobs DROP CONSTRAINT IF EXISTS processing_jobs_status_check;
ALTER TABLE processing_jobs
    ADD CONSTRAINT processing_jobs_status_check
    CHECK (status IN ('running','completed','failed'));
ALTER TABLE processing_jobs DROP COLUMN IF EXISTS duration_ms;
ALTER TABLE fits_files DROP COLUMN IF EXISTS skipped_reason;
ALTER TABLE fits_files DROP COLUMN IF EXISTS processing_ms;
