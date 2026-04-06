-- name: CreateEntry :one
INSERT INTO entries (
    user_id,
    recorded_at_utc,
    timezone_offset_minutes,
    entry_date,
    note_text,
    is_private,
    created_at_utc
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListEntries :many
SELECT
    e.id,
    e.user_id,
    e.recorded_at_utc,
    e.timezone_offset_minutes,
    e.entry_date,
    e.note_text,
    e.is_private,
    e.is_draft,
    e.audio_file_path,
    e.transcription_status,
    e.created_at_utc,
    tv.id AS trackable_value_id,
    tv.trackable_definition_id,
    tv.value_int,
    tv.value_bool,
    tv.value_text,
    tv.location_text,
    tv.note_text AS trackable_note_text,
    tv.entry_date AS trackable_entry_date,
    tv.created_at_utc AS trackable_created_at_utc,
    tv.updated_at_utc AS trackable_updated_at_utc,
    td.name AS trackable_name,
    td.icon AS trackable_icon,
    td.value_type AS trackable_value_type,
    td.unit AS trackable_unit,
    td.is_sensitive AS trackable_is_sensitive
FROM entries e
LEFT JOIN trackable_values tv ON tv.entry_id = e.id
LEFT JOIN trackable_definitions td ON td.id = tv.trackable_definition_id
WHERE e.user_id = sqlc.arg(user_id)
    AND (CAST(sqlc.arg(entry_date) AS TEXT) = '' OR e.entry_date = CAST(sqlc.arg(entry_date) AS TEXT))
ORDER BY e.recorded_at_utc DESC, tv.created_at_utc ASC;

-- name: GetEntryByID :one
SELECT *
FROM entries
WHERE id = ? AND user_id = ?;

-- name: GetEntryWithTrackables :many
SELECT
    e.id,
    e.user_id,
    e.recorded_at_utc,
    e.timezone_offset_minutes,
    e.entry_date,
    e.note_text,
    e.is_private,
    e.is_draft,
    e.audio_file_path,
    e.transcription_status,
    e.created_at_utc,
    tv.id AS trackable_value_id,
    tv.trackable_definition_id,
    tv.value_int,
    tv.value_bool,
    tv.value_text,
    tv.location_text,
    tv.note_text AS trackable_note_text,
    tv.entry_date AS trackable_entry_date,
    tv.created_at_utc AS trackable_created_at_utc,
    tv.updated_at_utc AS trackable_updated_at_utc,
    td.name AS trackable_name,
    td.icon AS trackable_icon,
    td.value_type AS trackable_value_type,
    td.unit AS trackable_unit,
    td.is_sensitive AS trackable_is_sensitive
FROM entries e
LEFT JOIN trackable_values tv ON tv.entry_id = e.id
LEFT JOIN trackable_definitions td ON td.id = tv.trackable_definition_id
WHERE e.user_id = ? AND e.id = ?
ORDER BY tv.created_at_utc ASC;

-- name: DeleteEntry :exec
DELETE FROM entries
WHERE id = ? AND user_id = ?;

-- name: UpdateEntryText :one
UPDATE entries
SET note_text = ?,
    is_private = ?
WHERE id = ? AND user_id = ?
RETURNING *;

-- name: CreateDraftEntry :one
INSERT INTO entries (
    user_id,
    recorded_at_utc,
    timezone_offset_minutes,
    entry_date,
    note_text,
    is_private,
    is_draft,
    audio_file_path,
    transcription_status,
    created_at_utc
)
VALUES (?, ?, ?, ?, NULL, 0, 1, ?, 'pending', ?)
RETURNING *;

-- name: UpdateEntryTranscription :exec
UPDATE entries
SET note_text = ?,
    is_draft = 0,
    transcription_status = 'completed'
WHERE id = ?;

-- name: MarkTranscriptionFailed :exec
UPDATE entries
SET transcription_status = 'failed'
WHERE id = ?;

-- name: InsertEntryImage :one
INSERT INTO entry_images (
    entry_id,
    user_id,
    file_path,
    mime_type,
    original_size,
    storage_tier,
    created_at_utc
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetImagesByEntryID :many
SELECT *
FROM entry_images
WHERE entry_id = ? AND user_id = ?
ORDER BY created_at_utc ASC;

-- name: GetImageByID :one
SELECT *
FROM entry_images
WHERE id = ? AND user_id = ?;

-- name: DeleteEntryImage :exec
DELETE FROM entry_images
WHERE id = ? AND user_id = ?;

-- name: DeleteImagesByEntryID :exec
DELETE FROM entry_images
WHERE entry_id = ? AND user_id = ?;

-- name: ListPendingTranscriptions :many
SELECT id, audio_file_path
FROM entries
WHERE transcription_status = 'pending'
AND audio_file_path IS NOT NULL;

-- name: GetTrackableTemplates :many
SELECT tt.*
FROM trackable_templates tt
LEFT JOIN trackable_definitions td
        ON td.template_id = tt.id
     AND td.user_id = ?
     AND td.deleted_at_utc IS NULL
WHERE td.id IS NULL;

-- name: GetAvailableTrackableTemplateByID :one
SELECT tt.*
FROM trackable_templates tt
LEFT JOIN trackable_definitions td
        ON td.template_id = tt.id
     AND td.user_id = ?
     AND td.deleted_at_utc IS NULL
WHERE tt.id = ?
    AND td.id IS NULL;

-- name: CreateTrackableDefinition :one
INSERT INTO trackable_definitions (
    user_id,
    template_id,
    name,
    value_type,
    unit,
    min_value,
    max_value,
    icon,
    category,
    is_sensitive,
    private_label,
    created_at_utc
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTrackableDefinitions :many
SELECT *
FROM trackable_definitions
WHERE user_id = ?;

-- name: ListTrackableDefinitionsWithDismissal :many
SELECT
    td.id,
    td.user_id,
    td.template_id,
    td.name,
    td.icon,
    td.value_type,
    td.unit,
    td.min_value,
    td.max_value,
    td.is_sensitive,
    td.private_label,
    td.category,
    td.active,
    td.created_at_utc,
    COALESCE(tdd.dismissed, 0) AS dismissed_today
FROM trackable_definitions td
LEFT JOIN trackable_daily_dismissals tdd
       ON tdd.trackable_definition_id = td.id
      AND tdd.user_id = td.user_id
      AND tdd.dismissal_date = ?
WHERE td.user_id = ?
    AND td.deleted_at_utc IS NULL
ORDER BY td.name;

-- name: UpsertTrackableDailyDismissal :one
INSERT INTO trackable_daily_dismissals (
    user_id,
    trackable_definition_id,
    dismissal_date,
    dismissed,
    created_at_utc,
    updated_at_utc
)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, trackable_definition_id, dismissal_date)
DO UPDATE SET
    dismissed = excluded.dismissed,
    updated_at_utc = excluded.updated_at_utc
RETURNING *;

-- name: CreateTrackableValue :one
INSERT INTO trackable_values (
    entry_id,
    trackable_definition_id,
    value_int,
    value_bool,
    value_text,
    created_at_utc
)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, entry_id, trackable_definition_id, value_int, value_bool, value_text, location_text, note_text, entry_date, created_at_utc, updated_at_utc;

-- name: FindRecentTrackableValue :one
SELECT tv.id, tv.entry_id, tv.trackable_definition_id, tv.value_int, tv.value_bool, tv.value_text, tv.location_text, tv.note_text, tv.entry_date, tv.created_at_utc, tv.updated_at_utc
FROM trackable_values tv
JOIN entries e ON e.id = tv.entry_id
WHERE tv.trackable_definition_id = ?
  AND e.user_id = ?
  AND tv.created_at_utc >= ?
ORDER BY tv.created_at_utc DESC
LIMIT 1;

-- name: FindEntryTrackableValue :one
SELECT tv.id, tv.entry_id, tv.trackable_definition_id, tv.value_int, tv.value_bool, tv.value_text, tv.location_text, tv.note_text, tv.entry_date, tv.created_at_utc, tv.updated_at_utc
FROM trackable_values tv
JOIN entries e ON e.id = tv.entry_id
WHERE tv.entry_id = ?
    AND tv.trackable_definition_id = ?
    AND e.user_id = ?
ORDER BY tv.created_at_utc DESC
LIMIT 1;

-- name: UpdateTrackableValueInt :one
UPDATE trackable_values
SET value_int = ?, updated_at_utc = ?
WHERE id = ?
RETURNING id, entry_id, trackable_definition_id, value_int, value_bool, value_text, location_text, note_text, entry_date, created_at_utc, updated_at_utc;

-- name: UpdateTrackableValueText :one
UPDATE trackable_values
SET value_text = ?, updated_at_utc = ?
WHERE id = ?
RETURNING id, entry_id, trackable_definition_id, value_int, value_bool, value_text, location_text, note_text, entry_date, created_at_utc, updated_at_utc;

-- name: GetTrackableById :one
SELECT *
FROM trackable_definitions
WHERE id = ? AND user_id = ? AND deleted_at_utc IS NULL;

-- name: SoftDeleteTrackableDefinition :exec
UPDATE trackable_definitions
SET deleted_at_utc = sqlc.arg(deleted_at_utc)
WHERE id = sqlc.arg(id)
    AND user_id = sqlc.arg(user_id)
    AND deleted_at_utc IS NULL;

-- name: ListSettings :many
SELECT *
FROM settings
WHERE user_id = ?;

-- name: GetSetting :one
SELECT *
FROM settings
WHERE user_id = sqlc.arg(user_id)
    AND settings_key = sqlc.arg(settings_key);

-- name: ListEntriesByUser :many
SELECT *
FROM entries
WHERE user_id = ?
ORDER BY recorded_at_utc DESC;

-- name: ListTrackableValuesByUser :many
SELECT
    tv.id,
    tv.entry_id,
    tv.trackable_definition_id,
    tv.value_int,
    tv.value_bool,
    tv.value_text,
    tv.location_text,
    tv.note_text,
    tv.entry_date,
    tv.created_at_utc,
    tv.updated_at_utc
FROM trackable_values tv
JOIN entries e ON e.id = tv.entry_id
WHERE e.user_id = ?
ORDER BY tv.created_at_utc DESC;

-- name: ListTrackableDailyDismissalsByUser :many
SELECT *
FROM trackable_daily_dismissals
WHERE user_id = ?
ORDER BY dismissal_date DESC;

-- name: ListWebauthnCredentialsByUser :many
SELECT *
FROM webauthn_credentials
WHERE user_id = ?
ORDER BY created_at_utc DESC;

-- name: CreateUser :one
INSERT INTO users (
    created_at_utc,
    webauthn_user_id,
    display_name
)
VALUES (?, ?, ?)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = ?;

-- name: GetUserByWebauthnUserID :one
SELECT *
FROM users
WHERE webauthn_user_id = ?;

-- name: CreateWebauthnCredential :one
INSERT INTO webauthn_credentials (
    user_id,
    credential_id,
    public_key,
    sign_count,
    flags,
    transports,
    created_at_utc
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateWebauthnCredentialSignCount :exec
UPDATE webauthn_credentials
SET sign_count = ?
WHERE credential_id = ?;

-- name: CreateDeviceLinkToken :one
INSERT INTO device_link_tokens (
        token_hash,
        user_id,
        expires_at_utc,
        created_at_utc
)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: RedeemDeviceLinkToken :one
UPDATE device_link_tokens
SET redeemed_at_utc = sqlc.arg(redeemed_at_utc)
WHERE token_hash = sqlc.arg(token_hash)
    AND expires_at_utc > sqlc.arg(now_utc)
    AND redeemed_at_utc IS NULL
    AND used_at_utc IS NULL
RETURNING *;

-- name: MarkDeviceLinkTokenUsed :execrows
UPDATE device_link_tokens
SET used_at_utc = sqlc.arg(used_at_utc)
WHERE id = sqlc.arg(token_id)
    AND user_id = sqlc.arg(user_id)
    AND redeemed_at_utc IS NOT NULL
    AND used_at_utc IS NULL;

-- name: UpsertSetting :exec
INSERT INTO settings (
    user_id,
    settings_key,
    settings_value,
    created_at_utc
)
VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, settings_key)
DO UPDATE SET settings_value = excluded.settings_value;

-- name: DeleteTrackableValuesByUser :exec
DELETE FROM trackable_values
WHERE entry_id IN (
    SELECT id
    FROM entries
    WHERE entries.user_id = sqlc.arg(target_user_id)
)
OR trackable_definition_id IN (
    SELECT id
    FROM trackable_definitions
    WHERE trackable_definitions.user_id = sqlc.arg(target_user_id)
);

-- name: DeleteTrackableDailyDismissalsByUser :exec
DELETE FROM trackable_daily_dismissals
WHERE user_id = ?;

-- name: DeleteEntriesByUser :exec
DELETE FROM entries
WHERE user_id = ?;

-- name: DeleteTrackableDefinitionsByUser :exec
DELETE FROM trackable_definitions
WHERE user_id = ?;

-- name: DeleteSettingsByUser :exec
DELETE FROM settings
WHERE user_id = ?;

-- name: DeleteWebauthnCredentialsByUser :exec
DELETE FROM webauthn_credentials
WHERE user_id = ?;
