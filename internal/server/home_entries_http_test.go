package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

func TestHomeRendersQuickCaptureOnly(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: yesterday, note: "yesterday note", recordedAtUTC: 1710000000})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, `action="/entry/add"`) {
		t.Fatalf("expected quick-capture form in home page, got: %s", body)
	}
	if !strings.Contains(body, `data-submit-state-button`) {
		t.Fatalf("expected submit-state button in home quick-capture form")
	}
	if strings.Contains(body, "yesterday note") {
		t.Fatalf("did not expect entries list content on home page")
	}
}

func TestEntriesRendersDayNavigationAndFiltersBySelectedDay(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: today, note: "today timeline note", recordedAtUTC: 1710000200})
	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: yesterday, note: "old timeline note", recordedAtUTC: 1710000000})

	req := httptest.NewRequest(http.MethodGet, "/entries?day="+today, nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "entries-day-nav") {
		t.Fatalf("expected day navigation to render in entries page")
	}
	if !strings.Contains(body, "today timeline note") {
		t.Fatalf("expected entry for selected day to be rendered")
	}
	if strings.Contains(body, "old timeline note") {
		t.Fatalf("did not expect entries outside selected day to be rendered")
	}
}

func TestEntriesAPIReturnsAudioFilePath(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")

	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: today, note: "typed note", recordedAtUTC: 1710000000})

	_, err := s.queries.CreateDraftEntry(context.Background(), db.CreateDraftEntryParams{
		UserID:                1,
		RecordedAtUtc:         1710000100,
		TimezoneOffsetMinutes: 0,
		EntryDate:             today,
		AudioFilePath:         sql.NullString{String: "/tmp/audio_1.webm", Valid: true},
		CreatedAtUtc:          1710000100,
	})
	if err != nil {
		t.Fatalf("create draft entry fixture: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/entries?day="+today, nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	var payload struct {
		Entries []struct {
			ID            int64   `json:"id"`
			IsDraft       bool    `json:"is_draft"`
			AudioFilePath *string `json:"audio_file_path"`
		} `json:"entries"`
		SelectedDay string `json:"selected_day"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/entries response: %v", err)
	}

	if payload.SelectedDay != today {
		t.Fatalf("expected selected_day=%q, got %q", today, payload.SelectedDay)
	}

	if len(payload.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(payload.Entries))
	}

	var foundDraft bool
	for _, entry := range payload.Entries {
		if !entry.IsDraft {
			continue
		}
		foundDraft = true
		if entry.AudioFilePath == nil || *entry.AudioFilePath != "/tmp/audio_1.webm" {
			t.Fatalf("expected draft entry audio_file_path to be set, got %#v", entry.AudioFilePath)
		}
	}

	if !foundDraft {
		t.Fatalf("expected to find a draft entry in API response")
	}
}

func TestTrackablesRouteRendersPicker(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/trackables", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), `data-trackable-picker`) {
		t.Fatalf("expected trackable picker content in /trackables response")
	}
}

func TestSettingsPostRedirectsAndPersists(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("language=no&theme=dark&screen_lock=300&share_timer=600")
	req := httptest.NewRequest(http.MethodPost, "/settings", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Location"); got != "/settings" {
		t.Fatalf("expected redirect to /settings, got %q", got)
	}

	settings, err := s.loadUserSettings(context.Background(), 1)
	if err != nil {
		t.Fatalf("loadUserSettings failed: %v", err)
	}
	if settings.Language != "no" || settings.Theme != "dark" || settings.ScreenLock != "300" || settings.ShareTimer != "600" {
		t.Fatalf("unexpected persisted settings: %+v", settings)
	}
}

func TestSettingsPostInvalidLanguageReturnsBadRequest(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("language=xx&theme=dark&screen_lock=300&share_timer=600")
	req := httptest.NewRequest(http.MethodPost, "/settings", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBottomNavActiveStateByRoute(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	tests := []struct {
		path         string
		expectedHref string
	}{
		{path: "/", expectedHref: "/"},
		{path: "/entries", expectedHref: "/entries"},
		{path: "/trackables", expectedHref: "/trackables"},
		{path: "/settings", expectedHref: "/settings"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
			}

			body := rr.Body.String()
			hrefs := []string{"/", "/entries", "/trackables", "/settings"}
			for _, href := range hrefs {
				snippet := `href="` + href + `" class="bottom-nav__link" aria-current="page"`
				if href == tt.expectedHref {
					if !strings.Contains(body, snippet) {
						t.Fatalf("expected active nav link %q in body", href)
					}
					continue
				}
				if strings.Contains(body, snippet) {
					t.Fatalf("did not expect active nav link %q for path %q", href, tt.path)
				}
			}
		})
	}
}

func TestTemplatesUseSavedUserLocale(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	err := s.saveUserSettings(context.Background(), 1, UserSettings{
		Language:   "no",
		Theme:      "system",
		ScreenLock: "none",
		ShareTimer: "300",
	}, 1710000000)
	if err != nil {
		t.Fatalf("saveUserSettings failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="no"`) {
		t.Fatalf("expected html lang to match user locale, body: %s", body)
	}
	if !strings.Contains(body, "Innstillinger") {
		t.Fatalf("expected Norwegian translation in response body")
	}
}

type entryFixture struct {
	userID        int64
	entryDate     string
	note          string
	recordedAtUTC int64
}

func newHomeEntriesHTTPTestServer(t *testing.T) *Server {
	t.Helper()

	root := projectRoot(t)

	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	schemaBytes, err := os.ReadFile(filepath.Join(root, "db", "schema.sql"))
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}

	if _, err := conn.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("apply schema.sql: %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO users (id, created_at_utc, webauthn_user_id, display_name, timezone) VALUES (1, 0, X'01', 'Test User', 'UTC')`); err != nil {
		t.Fatalf("seed test user: %v", err)
	}

	tmpl, err := parseTemplatesFromRoot(root, i18n.DefaultLocale)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	templatesByLocale := make(map[string]*template.Template)
	for _, locale := range i18n.Locales() {
		localized, err := parseTemplatesFromRoot(root, locale)
		if err != nil {
			t.Fatalf("parse templates for locale %q: %v", locale, err)
		}
		templatesByLocale[locale] = localized
	}

	s := &Server{
		mux:               http.NewServeMux(),
		templates:         tmpl,
		templatesByLocale: templatesByLocale,
		dbConn:            conn,
		devMode:           false,
		queries:           db.New(conn),
	}
	s.routes()

	return s
}

func insertEntryFixture(t *testing.T, s *Server, f entryFixture) {
	t.Helper()

	_, err := s.queries.CreateEntry(context.Background(), db.CreateEntryParams{
		UserID:                f.userID,
		RecordedAtUtc:         f.recordedAtUTC,
		TimezoneOffsetMinutes: 0,
		EntryDate:             f.entryDate,
		NoteText:              sql.NullString{String: f.note, Valid: f.note != ""},
		IsPrivate:             0,
		CreatedAtUtc:          f.recordedAtUTC,
	})
	if err != nil {
		t.Fatalf("create entry fixture: %v", err)
	}
}

func parseTemplatesFromRoot(root, locale string) (*template.Template, error) {
	tmpl := template.New("").Funcs(templateFuncMap(locale))

	viewsDir := filepath.Join(root, "internal", "views")
	var files []string
	err := filepath.WalkDir(viewsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return tmpl.ParseFiles(files...)
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
