-- +goose Up
-- +goose StatementBegin
ALTER TABLE server_configs DROP COLUMN server_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE server_configs ADD COLUMN server_id INT;
-- +goose StatementEnd
