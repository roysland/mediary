package server

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

type migration struct {
	id   string
	name string
	up   func(tx *sql.Tx) error
}

var migrations = []migration{
	{
		id:   "001_base_schema",
		name: "create base schema from db/schema.sql",
		up: func(tx *sql.Tx) error {
			schema, err := os.ReadFile("db/schema.sql")
			if err != nil {
				return fmt.Errorf("read schema.sql: %w", err)
			}
			if _, err := tx.Exec(string(schema)); err != nil {
				return fmt.Errorf("apply schema.sql: %w", err)
			}
			return nil
		},
	},
	{
		id:   "002_migrate_user_settings",
		name: "move legacy user_settings rows into settings",
		up: func(tx *sql.Tx) error {
			hasLegacy, err := tableExistsTx(tx, "user_settings")
			if err != nil {
				return err
			}
			if !hasLegacy {
				return nil
			}

			hasSettings, err := tableExistsTx(tx, "settings")
			if err != nil {
				return err
			}
			if !hasSettings {
				if _, err := tx.Exec(`
					CREATE TABLE IF NOT EXISTS settings (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						user_id INTEGER NOT NULL,
						settings_key TEXT NOT NULL,
						settings_value TEXT,
						created_at_utc INTEGER NOT NULL,
						FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
						UNIQUE(user_id, settings_key)
					)
				`); err != nil {
					return fmt.Errorf("create settings table: %w", err)
				}
			}

			if _, err := tx.Exec(`
				INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc)
				SELECT user_id, settings_key, settings_value, created_at_utc
				FROM user_settings
				ON CONFLICT(user_id, settings_key)
				DO UPDATE SET settings_value = excluded.settings_value
			`); err != nil {
				return fmt.Errorf("copy user_settings to settings: %w", err)
			}

			if _, err := tx.Exec(`DROP TABLE user_settings`); err != nil {
				return fmt.Errorf("drop legacy user_settings table: %w", err)
			}

			return nil
		},
	},
	{
		id:   "003_entries_entry_date",
		name: "ensure entries.entry_date exists",
		up: func(tx *sql.Tx) error {
			hasEntries, err := tableExistsTx(tx, "entries")
			if err != nil {
				return err
			}
			if !hasEntries {
				return nil
			}

			hasEntryDate, err := columnExistsTx(tx, "entries", "entry_date")
			if err != nil {
				return err
			}
			if hasEntryDate {
				return nil
			}

			hasLocalDate, err := columnExistsTx(tx, "entries", "local_date")
			if err != nil {
				return err
			}
			if hasLocalDate {
				if _, err := tx.Exec(`ALTER TABLE entries RENAME COLUMN local_date TO entry_date`); err != nil {
					return fmt.Errorf("rename entries.local_date to entry_date: %w", err)
				}
				return nil
			}

			if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN entry_date TEXT`); err != nil {
				return fmt.Errorf("add entries.entry_date: %w", err)
			}
			if _, err := tx.Exec(`
				UPDATE entries
				SET entry_date = date(recorded_at_utc, 'unixepoch')
				WHERE entry_date IS NULL
			`); err != nil {
				return fmt.Errorf("backfill entries.entry_date: %w", err)
			}

			return nil
		},
	},
	{
		id:   "004_trackable_values_updated_at",
		name: "ensure trackable_values.updated_at_utc exists",
		up: func(tx *sql.Tx) error {
			hasValues, err := tableExistsTx(tx, "trackable_values")
			if err != nil {
				return err
			}
			if !hasValues {
				return nil
			}

			hasCol, err := columnExistsTx(tx, "trackable_values", "updated_at_utc")
			if err != nil {
				return err
			}
			if hasCol {
				return nil
			}

			if _, err := tx.Exec(`ALTER TABLE trackable_values ADD COLUMN updated_at_utc INTEGER`); err != nil {
				return fmt.Errorf("add trackable_values.updated_at_utc: %w", err)
			}
			return nil
		},
	},
	{
		id:   "005_trackable_values_entry_date",
		name: "ensure trackable_values.entry_date exists",
		up: func(tx *sql.Tx) error {
			hasValues, err := tableExistsTx(tx, "trackable_values")
			if err != nil {
				return err
			}
			if !hasValues {
				return nil
			}

			hasCol, err := columnExistsTx(tx, "trackable_values", "entry_date")
			if err != nil {
				return err
			}
			if hasCol {
				return nil
			}

			if _, err := tx.Exec(`ALTER TABLE trackable_values ADD COLUMN entry_date TEXT`); err != nil {
				return fmt.Errorf("add trackable_values.entry_date: %w", err)
			}
			return nil
		},
	},
	{
		id:   "006_voice_entry_columns",
		name: "add voice/draft columns to entries",
		up: func(tx *sql.Tx) error {
			hasEntries, err := tableExistsTx(tx, "entries")
			if err != nil {
				return err
			}
			if !hasEntries {
				return nil
			}

			hasDraft, err := columnExistsTx(tx, "entries", "is_draft")
			if err != nil {
				return err
			}
			if !hasDraft {
				if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN is_draft INTEGER NOT NULL DEFAULT 0`); err != nil {
					return fmt.Errorf("add entries.is_draft: %w", err)
				}
			}

			hasAudio, err := columnExistsTx(tx, "entries", "audio_file_path")
			if err != nil {
				return err
			}
			if !hasAudio {
				if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN audio_file_path TEXT`); err != nil {
					return fmt.Errorf("add entries.audio_file_path: %w", err)
				}
			}

			hasStatus, err := columnExistsTx(tx, "entries", "transcription_status")
			if err != nil {
				return err
			}
			if !hasStatus {
				if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN transcription_status TEXT NOT NULL DEFAULT 'none'`); err != nil {
					return fmt.Errorf("add entries.transcription_status: %w", err)
				}
			}

			return nil
		},
	},
}

func runMigrations(conn *sql.DB) error {
	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at_utc INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	applied, err := loadAppliedMigrations(conn)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.id] {
			continue
		}

		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", m.id, err)
		}

		if err := m.up(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s (%s): %w", m.id, m.name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (id, name, applied_at_utc) VALUES (?, ?, ?)`,
			m.id,
			m.name,
			time.Now().Unix(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.id, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.id, err)
		}
	}

	return nil
}

func loadAppliedMigrations(conn *sql.DB) (map[string]bool, error) {
	rows, err := conn.Query(`SELECT id FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}

	return applied, nil
}

func tableExistsTx(tx *sql.Tx, tableName string) (bool, error) {
	row := tx.QueryRow(`SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1`, tableName)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check table %s exists: %w", tableName, err)
	}
	return exists == 1, nil
}

func columnExistsTx(tx *sql.Tx, tableName, columnName string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return false, fmt.Errorf("read table info for %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("scan table info for %s: %w", tableName, err)
		}
		if name == columnName {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table info for %s: %w", tableName, err)
	}

	return false, nil
}
