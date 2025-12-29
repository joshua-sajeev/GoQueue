-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
    ALTER COLUMN locked_by TYPE BIGINT
    USING locked_by::BIGINT;

CREATE INDEX IF NOT EXISTS idx_jobs_locked_at
    ON jobs(locked_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_jobs_locked_at;

ALTER TABLE jobs
    ALTER COLUMN locked_by TYPE VARCHAR(255);
-- +goose StatementEnd
