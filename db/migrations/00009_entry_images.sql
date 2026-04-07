-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS entry_images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    original_size INTEGER NOT NULL,
    storage_tier TEXT NOT NULL DEFAULT 'local',
    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (mime_type IN ('image/jpeg','image/png','image/webp','image/gif')),
    CHECK (storage_tier IN ('local','object'))
);

CREATE INDEX IF NOT EXISTS idx_entry_images_entry
ON entry_images(entry_id);

CREATE INDEX IF NOT EXISTS idx_entry_images_user
ON entry_images(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_entry_images_user;
DROP INDEX IF EXISTS idx_entry_images_entry;
DROP TABLE IF EXISTS entry_images;
-- +goose StatementEnd
