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
    td.unit AS trackable_unit
FROM entries e
LEFT JOIN trackable_values tv ON tv.entry_id = e.id
LEFT JOIN trackable_definitions td ON td.id = tv.trackable_definition_id
WHERE e.user_id = ?
ORDER BY e.recorded_at_utc DESC, tv.created_at_utc ASC;

-- name: ListEntriesByDay :many
SELECT
    e.id,
    e.user_id,
    e.recorded_at_utc,
    e.timezone_offset_minutes,
    e.entry_date,
    e.note_text,
    e.is_private,
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
    td.unit AS trackable_unit
FROM entries e
LEFT JOIN trackable_values tv ON tv.entry_id = e.id
LEFT JOIN trackable_definitions td ON td.id = tv.trackable_definition_id
WHERE e.user_id = ? AND e.entry_date = ?
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
    td.unit AS trackable_unit
FROM entries e
LEFT JOIN trackable_values tv ON tv.entry_id = e.id
LEFT JOIN trackable_definitions td ON td.id = tv.trackable_definition_id
WHERE e.user_id = ? AND e.id = ?
ORDER BY tv.created_at_utc ASC;

-- name: DeleteEntry :exec
DELETE FROM entries
WHERE id = ? AND user_id = ?;

-- name: GetTrackableTemplates :many
SELECT tt.*
FROM trackable_templates tt
LEFT JOIN trackable_definitions td
        ON td.template_id = tt.id
     AND td.user_id = ?
WHERE td.id IS NULL;

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
WHERE id = ? AND user_id = ?;

-- name: ListSettings :many
SELECT *
FROM settings
WHERE user_id = ?;

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

