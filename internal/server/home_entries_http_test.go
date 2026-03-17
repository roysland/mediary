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

func TestHomeRendersQuickCaptureAndTodayEntries(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: today, note: "today note", recordedAtUTC: 1710000100})
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
	if !strings.Contains(body, "/entries?day="+today) {
		t.Fatalf("expected home link to current entries day %q", today)
	}
	if !strings.Contains(body, "today note") {
		t.Fatalf("expected today's entry note to be shown on home page")
	}
	if strings.Contains(body, "yesterday note") {
		t.Fatalf("did not expect non-today entry note on home page")
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
	if !strings.Contains(body, "day-control") {
		t.Fatalf("expected day navigation to render in entries page")
	}
	if !strings.Contains(body, "today timeline note") {
		t.Fatalf("expected entry for selected day to be rendered")
	}
	if strings.Contains(body, "old timeline note") {
		t.Fatalf("did not expect entries outside selected day to be rendered")
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

	tmpl, err := parseTemplatesFromRoot(root)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	s := &Server{
		mux:       http.NewServeMux(),
		templates: tmpl,
		devMode:   false,
		queries:   db.New(conn),
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

func parseTemplatesFromRoot(root string) (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"t": i18n.T,
		"formatUnix": func(ts int64) string {
			return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04:05")
		},
		"formatISO": func(ts int64) string {
			return time.Unix(ts, 0).UTC().Format(time.RFC3339)
		},
		"json": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("null")
			}
			return template.JS(b)
		},
	})

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
