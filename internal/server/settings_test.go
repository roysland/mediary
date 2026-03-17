package server

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"roysland.me/symptomstracker/internal/db"
)

func TestDefaultUserSettings(t *testing.T) {
	got := defaultUserSettings()
	want := UserSettings{
		Language:   "en",
		Theme:      "system",
		ScreenLock: "none",
		ShareTimer: "300",
	}

	if got != want {
		t.Fatalf("unexpected defaults: got %+v want %+v", got, want)
	}
}

func TestLoadAndSaveUserSettings(t *testing.T) {
	s := newTestServerWithSettingsDB(t)
	ctx := context.Background()
	const userID int64 = 1
	const now int64 = 1710000000

	toSave := UserSettings{
		Language:   "no",
		Theme:      "dark",
		ScreenLock: "300",
		ShareTimer: "600",
	}

	if err := s.saveUserSettings(ctx, userID, toSave, now); err != nil {
		t.Fatalf("saveUserSettings failed: %v", err)
	}

	got, err := s.loadUserSettings(ctx, userID)
	if err != nil {
		t.Fatalf("loadUserSettings failed: %v", err)
	}

	if got != toSave {
		t.Fatalf("unexpected settings after roundtrip: got %+v want %+v", got, toSave)
	}
}

func TestLoadUserSettingsUsesDefaultsWhenMissing(t *testing.T) {
	s := newTestServerWithSettingsDB(t)
	ctx := context.Background()

	got, err := s.loadUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("loadUserSettings failed: %v", err)
	}

	if got != defaultUserSettings() {
		t.Fatalf("expected defaults for empty settings, got %+v", got)
	}
}

func newTestServerWithSettingsDB(t *testing.T) *Server {
	t.Helper()

	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, created_at_utc INTEGER NOT NULL, webauthn_user_id BLOB NOT NULL UNIQUE, display_name TEXT, timezone TEXT);`,
		`CREATE TABLE settings (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, settings_key TEXT NOT NULL, settings_value TEXT, created_at_utc INTEGER NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE, UNIQUE(user_id, settings_key));`,
		`INSERT INTO users (id, created_at_utc, webauthn_user_id, display_name, timezone) VALUES (1, 0, X'01', 'Test User', 'UTC');`,
	}
	for _, stmt := range stmts {
		if _, err := conn.Exec(stmt); err != nil {
			t.Fatalf("exec schema/setup statement failed: %v", err)
		}
	}

	return &Server{queries: db.New(conn), dbConn: conn}
}
