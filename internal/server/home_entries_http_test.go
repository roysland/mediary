package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"roysland.me/symptomstracker/internal/auth"
	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

func TestHomeRendersQuickCaptureOnly(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: yesterday, note: "yesterday note", recordedAtUTC: 1710000000})

	req := authedRequest(t, s, http.MethodGet, "/", nil)
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

	req := authedRequest(t, s, http.MethodGet, "/entries?day="+today, nil)
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

func TestEntriesRenderEntryDialogForPastDay(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	req := authedRequest(t, s, http.MethodGet, "/entries?day="+yesterday, nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, `id="entry-note-dialog"`) || !strings.Contains(body, `class="sheet"`) {
		t.Fatalf("expected note dialog sheet in entries page")
	}
	if !strings.Contains(body, `name="entry_date" value="`+yesterday+`"`) {
		t.Fatalf("expected selected day hidden in entry form, got: %s", body)
	}
	if !strings.Contains(body, `You are adding an entry to a past day.`) {
		t.Fatalf("expected past-day warning in note dialog")
	}
}

func TestEntriesRenderSensitiveFilterMarkers(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")

	privateEntry := insertEntryFixture(t, s, entryFixture{
		userID:        1,
		entryDate:     today,
		note:          "private note",
		recordedAtUTC: 1710000200,
		isPrivate:     1,
	})

	sensitiveTrackable := insertTrackableDefinitionFixture(t, s, trackableDefinitionFixture{
		userID:      1,
		name:        "Medication",
		valueType:   "text",
		icon:        "💊",
		category:    "symptom",
		isSensitive: 1,
	})

	insertTrackableValueFixture(t, s, trackableValueFixture{
		entryID:      privateEntry.ID,
		trackableID:  sensitiveTrackable.ID,
		valueText:    "Evening dose",
		createdAtUTC: 1710000300,
	})

	req := authedRequest(t, s, http.MethodGet, "/entries?day="+today, nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, `data-sensitive-filter-toggle`) {
		t.Fatalf("expected sensitive filter toggle in entries page")
	}
	if !strings.Contains(body, `Show private and sensitive`) {
		t.Fatalf("expected sensitive filter label in entries page")
	}
	if !strings.Contains(body, `data-entry-private="true"`) {
		t.Fatalf("expected private entries to be marked for filtering")
	}
	if !strings.Contains(body, `data-sensitive-trackable="true"`) {
		t.Fatalf("expected sensitive trackables to be marked for filtering")
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

	req := authedRequest(t, s, http.MethodGet, "/api/entries?day="+today, nil)
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

func TestAddEntryUsesProvidedEntryDate(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	body := strings.NewReader("entry_input=retro+note&entry_date=" + yesterday)
	req := authedRequest(t, s, http.MethodPost, "/entry/add", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Location"); got != "/entries?day="+yesterday {
		t.Fatalf("expected redirect to selected day, got %q", got)
	}

	entries, err := s.queries.ListEntriesByUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("list entries by user: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].EntryDate != yesterday {
		t.Fatalf("expected entry_date=%q, got %q", yesterday, entries[0].EntryDate)
	}
}

func TestEntriesRenderAddTextActionForTrackableOnlyEntry(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")
	entry := insertEntryFixture(t, s, entryFixture{
		userID:        1,
		entryDate:     today,
		recordedAtUTC: 1710000000,
	})
	trackable := insertTrackableDefinitionFixture(t, s, trackableDefinitionFixture{
		userID:    1,
		name:      "Energy",
		valueType: "text",
		icon:      "⚡",
		category:  "symptom",
	})
	insertTrackableValueFixture(t, s, trackableValueFixture{
		entryID:      entry.ID,
		trackableID:  trackable.ID,
		valueText:    "Low",
		createdAtUTC: 1710000100,
	})

	req := authedRequest(t, s, http.MethodGet, "/entries?day="+today, nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, `class="button--ghost button--sm edit-entry-button"`) {
		t.Fatalf("expected edit-entry action in context menu")
	}
	if !strings.Contains(body, `data-entry-has-note="false"`) || !strings.Contains(body, `Add text`) {
		t.Fatalf("expected add text action for trackable-only entry, got: %s", body)
	}
}

func TestEditEntryUpdatesExistingText(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")
	entry := insertEntryFixture(t, s, entryFixture{
		userID:        1,
		entryDate:     today,
		recordedAtUTC: 1710000000,
	})

	body := strings.NewReader("entry_input=filled+in+later&is_private_entry=on")
	req := authedRequest(t, s, http.MethodPost, "/entry/"+strconv.FormatInt(entry.ID, 10)+"/edit", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, err := s.queries.GetEntryByID(context.Background(), db.GetEntryByIDParams{
		ID:     entry.ID,
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("get updated entry: %v", err)
	}
	if !updated.NoteText.Valid || updated.NoteText.String != "filled in later" {
		t.Fatalf("expected updated note text, got %#v", updated.NoteText)
	}
	if updated.IsPrivate != 1 {
		t.Fatalf("expected private flag to be updated, got %d", updated.IsPrivate)
	}
}

func TestAddEntryWithEntryIDUpdatesExistingEntry(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")
	entry := insertEntryFixture(t, s, entryFixture{
		userID:        1,
		entryDate:     today,
		note:          "",
		recordedAtUTC: 1710000000,
	})

	body := strings.NewReader(
		"entry_id=" + strconv.FormatInt(entry.ID, 10) +
			"&entry_input=added+later" +
			"&entry_date=" + today +
			"&is_private_entry=on",
	)
	req := authedRequest(t, s, http.MethodPost, "/entry/add", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
	}

	entries, err := s.queries.ListEntriesByUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("list entries by user: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected entry to be updated in place, got %d entries", len(entries))
	}

	updated, err := s.queries.GetEntryByID(context.Background(), db.GetEntryByIDParams{
		ID:     entry.ID,
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("get updated entry: %v", err)
	}
	if !updated.NoteText.Valid || updated.NoteText.String != "added later" {
		t.Fatalf("expected updated note text, got %#v", updated.NoteText)
	}
	if updated.IsPrivate != 1 {
		t.Fatalf("expected private flag to be updated, got %d", updated.IsPrivate)
	}
}
func TestTrackablesRouteRendersPicker(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodGet, "/trackables", nil)
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
	req := authedRequest(t, s, http.MethodPost, "/settings", body)
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
	req := authedRequest(t, s, http.MethodPost, "/settings", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSettingsPostCrossSiteRejected(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("language=no&theme=dark&screen_lock=300&share_timer=600")
	req := authedRequest(t, s, http.MethodPost, "/settings", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSettingsPostSameOriginAllowed(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("language=no&theme=dark&screen_lock=300&share_timer=600")
	req := authedRequest(t, s, http.MethodPost, "/settings", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
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
			req := authedRequest(t, s, http.MethodGet, tt.path, nil)
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

	req := authedRequest(t, s, http.MethodGet, "/settings", nil)
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
	isPrivate     int64
}

type trackableDefinitionFixture struct {
	userID      int64
	name        string
	valueType   string
	icon        string
	category    string
	isSensitive int64
}

type trackableValueFixture struct {
	entryID      int64
	trackableID  int64
	valueText    string
	createdAtUTC int64
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
		cfg: Config{
			AuthSessionSecret: "0123456789abcdef0123456789abcdef",
		},
		ceremonies: make(map[string]webauthnCeremony),
	}
	authSessions, err := auth.NewSessionManager(s.cfg.AuthSessionSecret, false)
	if err != nil {
		t.Fatalf("create auth session manager: %v", err)
	}
	s.authSessions = authSessions
	auth.SetDefaultSessionManager(authSessions)
	s.routes()

	return s
}

func insertEntryFixture(t *testing.T, s *Server, f entryFixture) db.Entry {
	t.Helper()

	entry, err := s.queries.CreateEntry(context.Background(), db.CreateEntryParams{
		UserID:                f.userID,
		RecordedAtUtc:         f.recordedAtUTC,
		TimezoneOffsetMinutes: 0,
		EntryDate:             f.entryDate,
		NoteText:              sql.NullString{String: f.note, Valid: f.note != ""},
		IsPrivate:             f.isPrivate,
		CreatedAtUtc:          f.recordedAtUTC,
	})
	if err != nil {
		t.Fatalf("create entry fixture: %v", err)
	}

	return entry
}

func insertTrackableDefinitionFixture(t *testing.T, s *Server, f trackableDefinitionFixture) db.TrackableDefinition {
	t.Helper()

	definition, err := s.queries.CreateTrackableDefinition(context.Background(), db.CreateTrackableDefinitionParams{
		UserID:       f.userID,
		Name:         f.name,
		ValueType:    f.valueType,
		Icon:         sql.NullString{String: f.icon, Valid: f.icon != ""},
		Category:     f.category,
		IsSensitive:  f.isSensitive,
		CreatedAtUtc: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("create trackable fixture: %v", err)
	}

	return definition
}

func insertTrackableValueFixture(t *testing.T, s *Server, f trackableValueFixture) db.TrackableValue {
	t.Helper()

	value, err := s.queries.CreateTrackableValue(context.Background(), db.CreateTrackableValueParams{
		EntryID:               f.entryID,
		TrackableDefinitionID: f.trackableID,
		ValueText:             sql.NullString{String: f.valueText, Valid: f.valueText != ""},
		CreatedAtUtc:          f.createdAtUTC,
	})
	if err != nil {
		t.Fatalf("create trackable value fixture: %v", err)
	}

	return value
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

func authedRequest(t *testing.T, s *Server, method, target string, body io.Reader) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, body)

	rr := httptest.NewRecorder()
	if err := s.authSessions.SetAuthenticatedUser(rr, 1); err != nil {
		t.Fatalf("set authenticated user cookie: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	return req
}
