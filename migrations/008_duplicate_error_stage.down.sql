-- 008_duplicate_error_stage.down.sql

ALTER TABLE processing_errors DROP CONSTRAINT IF EXISTS processing_errors_stage_check;
ALTER TABLE processing_errors
    ADD CONSTRAINT processing_errors_stage_check
    CHECK (stage IN ('scan','parse','insert'));
