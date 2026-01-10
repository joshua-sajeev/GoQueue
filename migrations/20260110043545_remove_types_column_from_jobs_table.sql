-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
DROP COLUMN type;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs
ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT '';
-- +goose StatementEnd

