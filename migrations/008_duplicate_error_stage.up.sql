-- 008_duplicate_error_stage.up.sql
-- Allows a 'duplicate' stage in processing_errors so each detected duplicate
-- gets its own audit-trail row (which file was original, which was the dup),
-- in addition to the fits_files.status='skipped' + skipped_reason on the file itself.

ALTER TABLE processing_errors DROP CONSTRAINT IF EXISTS processing_errors_stage_check;
ALTER TABLE processing_errors
    ADD CONSTRAINT processing_errors_stage_check
    CHECK (stage IN ('scan','parse','insert','duplicate'));
