-- +goose Up
ALTER TABLE jar_files ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE additional_files ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- +goose Down
ALTER TABLE jar_files DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE additional_files DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE servers DROP COLUMN IF EXISTS deleted_at;