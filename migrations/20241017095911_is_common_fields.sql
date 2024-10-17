-- +goose Up
-- +goose StatementBegin
ALTER TABLE jar_files ADD COLUMN IF NOT EXISTS is_common BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE additional_files ADD COLUMN IF NOT EXISTS is_common BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jar_files DROP COLUMN IF EXISTS is_common;
ALTER TABLE additional_files DROP COLUMN IF EXISTS is_common;
-- +goose StatementEnd