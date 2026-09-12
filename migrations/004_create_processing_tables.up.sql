-- 004_create_processing_tables.up.sql
-- Tracks each scan run and per-file errors.

CREATE TABLE IF NOT EXISTS processing_jobs (
    id           BIGSERIAL   PRIMARY KEY,
    scan_dir     TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'running'
                             CHECK (status IN ('running','completed','failed')),
    total_files  INT         NOT NULL DEFAULT 0,
    done_files   INT         NOT NULL DEFAULT 0,
    error_files  INT         NOT NULL DEFAULT 0,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at  TIMESTAMPTZ,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS processing_errors (
    id          BIGSERIAL   PRIMARY KEY,
    job_id      BIGINT      NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    file_id     BIGINT      REFERENCES fits_files(id) ON DELETE SET NULL,
    file_path   TEXT        NOT NULL,
    stage       TEXT        NOT NULL CHECK (stage IN ('scan','parse','insert')),
    message     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_proc_errors_job_id  ON processing_errors (job_id);
CREATE INDEX idx_proc_errors_file_id ON processing_errors (file_id);
