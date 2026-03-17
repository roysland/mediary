## Follow-up after i18n catalog refactor

- Thread the user's selected language into template rendering instead of always using the default locale.
	Why: `internal/i18n` now supports locale-aware lookup, but `internal/server/server.go` still registers `t` as a global English-only function, so choosing Norwegian in settings has no effect on rendered UI.
	How: Resolve the current locale per request or page view, expose a locale-bound translation helper to templates, and add a second locale catalog once real translations are available.

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

## Follow-ups after preset flow cleanup

- Decide whether editing preset-populated fields should clear `trackable_template_id` automatically.
	Why: The server now persists `template_id` when present, but the current client behavior only clears it when the name input changes, not when other preset-populated fields are edited.
	How: Either keep `template_id` sticky for all edits (explicitly treating presets as source templates) or clear it on changes to icon/type/category/min/max/unit/private-label for fully custom derivatives.

## Follow-ups after runtime artifact ignore cleanup

- Consider purging previously committed binaries/runtime files from git history using `git-filter-repo`.
	Why: `.gitignore` prevents new local artifacts from entering future diffs, but old commits can still keep repository size/noise if files like `server`, `tmp/app`, or `data/app.db` were committed earlier.
	How: In a coordinated branch, run `git filter-repo --path server --path tmp/app --path data/app.db --invert-paths`, then force-push and have collaborators re-clone or hard-reset after the history rewrite.

## Follow-ups after home quick-capture simplification

- Add a browser-level test for submit-state transitions on the home quick-capture form.
	Why: Server tests now verify that the `data-submit-state-button` hook exists, but they cannot verify the client-side loading/success visual cue behavior.
	How: Add a Playwright test that submits the `/` form with HTMX enabled and asserts the button transitions through `data-submit-visual="loading"` to `"success"`, then returns to `"idle"` while the form input is cleared.

