-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    queue VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL,
    payload JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    result JSONB,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX idx_jobs_status ON jobs(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs;
-- +goose StatementEnd
