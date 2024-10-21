-- +goose Up
-- +goose StatementBegin
CREATE TYPE server_status AS ENUM ('running', 'stopped');
ALTER TABLE servers ADD COLUMN user_id INTEGER;
ALTER TABLE servers ADD COLUMN status server_status DEFAULT 'stopped';
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN user_id;
ALTER TABLE servers DROP COLUMN status;
-- +goose StatementEnd
