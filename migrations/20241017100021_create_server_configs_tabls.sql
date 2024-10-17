-- +goose Up
CREATE TABLE server_configs (
    id SERIAL PRIMARY KEY,
    server_id INTEGER NOT NULL,
    jar_file_id INTEGER NOT NULL,
    mod_pack_id INTEGER,
    executable_command TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (jar_file_id) REFERENCES jar_files(id),
    FOREIGN KEY (mod_pack_id) REFERENCES mod_packs(id)
);

-- +goose Down
DROP TABLE server_configs;