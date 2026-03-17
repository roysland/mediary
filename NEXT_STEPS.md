## Follow-ups after i18n key coverage cleanup

- Localize remaining hardcoded UI strings in `internal/views/topNav.html`.
	Why: The menu and theme toggle labels are still English-only and not routed through i18n.
	How: Replace hardcoded `aria-label` and button text with `t "..."` keys and add matching entries in `internal/i18n/i18n.go`.

- Localize client-side confirmation/error strings in `web/static/entries.js` and fallback message in `web/static/settings.js`.
	Why: Delete and confirmation flows still show hardcoded English browser dialogs.
	How: Pass translated strings through data attributes from templates and read those values in JavaScript instead of inline literals.

## Follow-ups after handler normalization

- Extend `requireMethod`, `requirePathInt64`, and `requireParsedForm` usage to remaining server handlers for full consistency.
	Why: Task 17 now normalizes `entries`, `trackable`, and `settings`, but other handlers still use mixed direct checks and parsing styles.
	How: Audit `internal/server/*.go` for direct `r.Method`, `r.ParseForm`, and `strconv.ParseInt(r.PathValue(...))` patterns and migrate them to shared helpers where behavior matches.

## Follow-ups after constants cleanup

- Replace repeated date layout literals with shared constants.
	Why: Values like `"2006-01-02"` and related time formats are still duplicated across handlers/services and can drift over time.
	How: Add date/time layout constants in `internal/server/constants.go` and update parsing/formatting call sites to use them.

## Follow-ups after query deduplication

- Validate query plan and indexing for optional day filter in `ListEntries`.
	Why: The new optional predicate (`entry_date IS NULL OR e.entry_date = entry_date`) reduces SQL duplication but can lead to less predictable index usage as data grows.
	How: Run `EXPLAIN QUERY PLAN` for day-filtered and unfiltered variants, then add or tune a composite index (for example `(user_id, entry_date, recorded_at_utc)`) if scans become expensive.

## Follow-ups after baseline test coverage

- Add end-to-end handler tests for `entries`, `trackable`, and `settings` routes.
	Why: Current tests mainly cover helper and service-adjacent logic; route wiring, auth flow, and rendered HTTP behavior are still weakly covered.
	How: Use `httptest.NewRecorder` with a test `Server` backed by in-memory SQLite and exercise GET/POST handlers, asserting status codes, redirects, and key response snippets.
