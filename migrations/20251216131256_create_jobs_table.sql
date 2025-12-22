-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    queue VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL,
    payload JSONB,

    status VARCHAR(50) NOT NULL DEFAULT 'queued',

    attempts INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,

    available_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    locked_at TIMESTAMP WITH TIME ZONE,
    locked_by VARCHAR(255),

    result JSONB,
    error TEXT,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_queue_status ON jobs(queue, status);
CREATE INDEX idx_jobs_available_at ON jobs(available_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs;
-- +goose StatementEnd
