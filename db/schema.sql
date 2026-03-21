-- NOTE: This file is the canonical schema definition used by sqlc and base-schema migration.
-- Runtime schema changes are applied through tracked migrations in internal/server/migrations.go.

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at_utc INTEGER NOT NULL,

    -- WebAuthn identifiers
    webauthn_user_id BLOB NOT NULL UNIQUE,
    display_name TEXT,

    -- optional future settings
    timezone TEXT
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id INTEGER NOT NULL,
    credential_id BLOB NOT NULL UNIQUE,
    public_key BLOB NOT NULL,

    sign_count INTEGER NOT NULL,

    flags TEXT,
    transports TEXT,
    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS device_link_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    token_hash BLOB NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,

    expires_at_utc INTEGER NOT NULL,
    redeemed_at_utc INTEGER,
    used_at_utc INTEGER,
    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id INTEGER NOT NULL,

    recorded_at_utc INTEGER NOT NULL,
    timezone_offset_minutes INTEGER NOT NULL,

    entry_date TEXT NOT NULL, -- YYYY-MM-DD: user-specified date (may differ from recorded_at for retroactive entries)

    note_text TEXT,
    is_private INTEGER NOT NULL DEFAULT 0,

    is_draft INTEGER NOT NULL DEFAULT 0,
    audio_file_path TEXT,
    transcription_status TEXT NOT NULL DEFAULT 'none',

    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    settings_key TEXT NOT NULL,
    settings_value TEXT,
    created_at_utc INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, settings_key)
);

CREATE TABLE IF NOT EXISTS trackable_definitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id INTEGER NOT NULL,
    template_id INTEGER,

    name TEXT NOT NULL,
    icon TEXT,

    value_type TEXT NOT NULL, 
    -- 'integer', 'boolean', 'text'

    unit TEXT,

    min_value INTEGER,
    max_value INTEGER,

    is_sensitive INTEGER NOT NULL DEFAULT 0,
    private_label TEXT,
    category TEXT NOT NULL,

    active INTEGER NOT NULL DEFAULT 1,

    deleted_at_utc INTEGER, -- NULL = live, Unix timestamp = soft-deleted

    created_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES trackable_templates(id) ON DELETE SET NULL,
    CHECK (value_type IN ('integer', 'boolean', 'text')),
    CHECK (category IN ('default', 'symptom', 'activity', 'measurement', 'state'))
);

CREATE TABLE IF NOT EXISTS trackable_values (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    entry_id INTEGER NOT NULL,
    trackable_definition_id INTEGER NOT NULL,

    value_int INTEGER,
    value_bool INTEGER,
    value_text TEXT,

    location_text TEXT,

    note_text TEXT,

    entry_date TEXT, -- YYYY-MM-DD: optional retroactive date; NULL inherits from parent entry

    created_at_utc INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at_utc INTEGER,

    FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
    FOREIGN KEY (trackable_definition_id) REFERENCES trackable_definitions(id) ON DELETE CASCADE,
    UNIQUE(entry_id, trackable_definition_id),
    CHECK (
        (value_int IS NOT NULL) +
        (value_bool IS NOT NULL) +
        (value_text IS NOT NULL)
        = 1
    ),

    CHECK (value_bool IN (0,1) OR value_bool IS NULL)
);

CREATE TABLE IF NOT EXISTS trackable_daily_dismissals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id INTEGER NOT NULL,
    trackable_definition_id INTEGER NOT NULL,
    dismissal_date TEXT NOT NULL, -- YYYY-MM-DD
    dismissed INTEGER NOT NULL DEFAULT 1,
    created_at_utc INTEGER NOT NULL,
    updated_at_utc INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (trackable_definition_id) REFERENCES trackable_definitions(id) ON DELETE CASCADE,
    UNIQUE(user_id, trackable_definition_id, dismissal_date),
    CHECK (dismissed IN (0,1))
);


CREATE TABLE IF NOT EXISTS trackable_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    name TEXT NOT NULL UNIQUE,

    value_type TEXT NOT NULL,

    unit TEXT,

    min_value INTEGER,
    max_value INTEGER,

    icon TEXT,

    is_sensitive INTEGER NOT NULL DEFAULT 0,
    private_label TEXT,
    custom_control_type TEXT,
    category TEXT,

    created_at_utc INTEGER NOT NULL,

    CHECK (value_type IN ('integer', 'boolean', 'text'))
);

CREATE INDEX IF NOT EXISTS idx_trackable_templates_name
ON trackable_templates(name);

CREATE INDEX IF NOT EXISTS idx_trackable_definitions_template_id
ON trackable_definitions(template_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trackable_definitions_user_name_active
ON trackable_definitions(user_id, name)
WHERE deleted_at_utc IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_trackable_definitions_user_template
ON trackable_definitions(user_id, template_id)
WHERE template_id IS NOT NULL AND deleted_at_utc IS NULL;

CREATE INDEX IF NOT EXISTS idx_entries_user_date
ON entries(user_id, entry_date);

CREATE INDEX IF NOT EXISTS idx_entries_user_time
ON entries(user_id, recorded_at_utc);

CREATE INDEX IF NOT EXISTS idx_trackable_values_entry
ON trackable_values(entry_id);

CREATE INDEX IF NOT EXISTS idx_trackable_values_definition
ON trackable_values(trackable_definition_id);

CREATE INDEX IF NOT EXISTS idx_trackable_daily_dismissals_user_date
ON trackable_daily_dismissals(user_id, dismissal_date);

CREATE INDEX IF NOT EXISTS idx_device_link_tokens_user
ON device_link_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_device_link_tokens_expires
ON device_link_tokens(expires_at_utc);

INSERT OR IGNORE INTO trackable_templates
(name, value_type, unit, min_value, max_value, icon, is_sensitive, private_label, custom_control_type, category, created_at_utc)
VALUES
-- Core ME/CFS symptoms
('Fatigue', 'integer', NULL, 0, 10, '⚡', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Post-exertional malaise', 'integer', NULL, 0, 10, '🪫', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Brain fog', 'integer', NULL, 0, 10, '🧠', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Sleep quality', 'integer', NULL, 0, 10, '😴', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Pain', 'integer', NULL, 0, 10, '🔥', 0, NULL, NULL, 'symptom', strftime('%s','now')),

-- Cardiovascular
('Heart rate', 'integer', 'bpm', 30, 200, '❤️', 0, NULL, 'heart-rate', 'measurement', strftime('%s','now')),
('Blood pressure', 'text', 'mmHg', NULL, NULL, '🩺', 0, NULL, 'blood-pressure', 'measurement', strftime('%s','now')),
('Palpitations', 'boolean', NULL, NULL, NULL, '💓', 0, NULL, NULL, 'measurement', strftime('%s','now')),
('Dizziness', 'integer', NULL, 0, 10, '💫', 0, NULL, NULL, 'measurement', strftime('%s','now')),

-- Energy / activity
('Physical activity', 'integer', NULL, 0, 10, '🚶', 0, NULL, NULL, 'activity', strftime('%s','now')),
('Steps', 'integer', 'steps', 0, 10000, '👣', 0, NULL, NULL, 'activity', strftime('%s','now')),

-- Sleep
('Hours slept', 'integer', 'hours', 0, 24, '🛌', 0, NULL, NULL, 'measurement',strftime('%s','now')),
('Unrefreshing sleep', 'boolean', NULL, NULL, NULL, '🌙', 0, NULL, NULL, 'measurement',strftime('%s','now')),

-- Neurological / sensory
('Headache', 'integer', NULL, 0, 10, '🤕', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Light sensitivity', 'integer', NULL, 0, 10, '💡', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Sound sensitivity', 'integer', NULL, 0, 10, '🔊', 0, NULL, NULL, 'symptom', strftime('%s','now')),

-- Autonomic
('Temperature', 'integer', '°C', 30, 42, '🌡️', 0, NULL, NULL, 'measurement', strftime('%s','now')),
('Feeling feverish', 'boolean', NULL, NULL, NULL, '🥵', 0, NULL, NULL, 'symptom', strftime('%s','now')),

-- Digestive
('Nausea', 'integer', NULL, 0, 10, '🤢', 0, NULL, NULL, 'symptom', strftime('%s','now')),

-- Mood / mental
('Mood', 'integer', NULL, 0, 10, '🙂', 0, NULL, NULL, 'symptom', strftime('%s','now')),
('Anxiety', 'integer', NULL, 0, 10, '😟', 0, NULL, NULL, 'symptom', strftime('%s','now')),

-- Sensitive but important biological signals
('Sexual activity', 'boolean', NULL, NULL, NULL, '🔒', 1, 'Private activity', NULL, 'activity', strftime('%s','now'));

