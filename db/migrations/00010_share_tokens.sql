-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS share_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash BLOB NOT NULL UNIQUE,
    password_hash BLOB NOT NULL,
    scope_date_from TEXT,
    scope_date_to TEXT,
    scope_private INTEGER NOT NULL DEFAULT 0,
    expires_at_utc INTEGER NOT NULL,
    accessed_at_utc INTEGER,
    revoked_at_utc INTEGER,
    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (scope_private IN (0,1))
);

CREATE INDEX IF NOT EXISTS idx_share_tokens_user
ON share_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_share_tokens_expires
ON share_tokens(expires_at_utc);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_share_tokens_expires;
DROP INDEX IF EXISTS idx_share_tokens_user;
DROP TABLE IF EXISTS share_tokens;
-- +goose StatementEnd
