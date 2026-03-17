package server

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestListEntriesOptionalDayFilterUsesUserTimeIndex(t *testing.T) {
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

	query := `
		EXPLAIN QUERY PLAN
		SELECT *
		FROM entries e
		WHERE e.user_id = ?
		  AND (CAST(? AS TEXT) = '' OR e.entry_date = CAST(? AS TEXT))
		ORDER BY e.recorded_at_utc DESC
	`

	assertPlanUsesIndex := func(day string) {
		t.Helper()
		rows, err := conn.Query(query, int64(1), day, day)
		if err != nil {
			t.Fatalf("run explain query plan: %v", err)
		}
		defer rows.Close()

		details := make([]string, 0)
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
				t.Fatalf("scan explain row: %v", err)
			}
			details = append(details, detail)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate explain rows: %v", err)
		}

		plan := strings.Join(details, "\n")
		if !strings.Contains(plan, "idx_entries_user_time") {
			t.Fatalf("expected plan to include idx_entries_user_time, got:\n%s", plan)
		}
	}

	assertPlanUsesIndex("")
	assertPlanUsesIndex("2026-03-17")
}
