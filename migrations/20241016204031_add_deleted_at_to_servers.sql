-- +goose Up
ALTER TABLE servers ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
-- +goose Down
ALTER TABLE servers DROP COLUMN deleted_at;