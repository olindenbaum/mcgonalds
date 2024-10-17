-- +goose Up
-- +goose StatementBegin
ALTER TABLE server_configs
ADD COLUMN server_id INTEGER UNIQUE NOT NULL, -- Adjust data type if necessary
ADD CONSTRAINT fk_server_id FOREIGN KEY (server_id) REFERENCES servers (id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE server_configs DROP COLUMN server_id;
-- +goose StatementEnd
