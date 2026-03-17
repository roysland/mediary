# Agent Instruction: Revalidated NEXT_STEPS Backlog (2026-03-17)

Use this as the source of truth for follow-up work. Revalidation is based on current repository state.

## Scope

1. Keep only items marked `still-needed` or `partial` in active planning.
2. Remove or archive items marked `not-needed`.
3. Respect project constraints in `AGENTS.md` (never edit `internal/db/*`, do not edit `web/static/dist/*`).

## Revalidated Items

1. `still-needed` Thread selected language into template rendering.
Reason: Templates still bind global `t` to `i18n.T` (default English) in `internal/server/server.go:62`; locale-aware helper is not request-bound.
2. `still-needed` Add non-English locale catalog.
Reason: `internal/i18n/i18n.go:12` registers only `LocaleEnglish: englishCatalog`; no Norwegian catalog is loaded.
3. `still-needed` Localize hardcoded top-nav strings.
Reason: `internal/views/topNav.html:4` and `internal/views/topNav.html:10` still contain literal English labels (`Open menu`, `Menu`, `Switch theme`, `Theme`).
4. `still-needed` Localize client-side entry delete/alert strings.
Reason: `web/static/entries.js:2`, `web/static/entries.js:15`, and `web/static/entries.js:26` still use hardcoded English text.
5. `partial` Localize settings.js fallback confirm message.
Reason: Main confirmation already comes from translated `data-confirm-message`, but fallback `Are you sure?` remains in `web/static/settings.js:16`.
6. `not-needed` Extend `requireMethod`/`requirePathInt64`/`requireParsedForm` usage.
Reason: Remaining server handlers already use shared helpers; direct path parsing/form parsing patterns are confined to `internal/server/http_helpers.go`.
7. `still-needed` Replace repeated date layout literals with constants.
Reason: Date/time layouts are duplicated across runtime paths (for example `internal/server/entry_service.go:25`, `internal/server/entries.go:31`, `internal/server/home.go:18`, `internal/server/server.go:64`).
8. `still-needed` Validate `ListEntries` query plan/index behavior for optional day filter.
Reason: Optional filter still exists in `db/queries.sql:39`; indexes exist but no committed query-plan validation evidence.
9. `partial` Add route-level handler tests for entries, trackable, settings.
Reason: `internal/server/home_entries_http_test.go` covers `/` and `/entries` GET rendering, but no equivalent route tests for `/trackables` or `/settings` POST/redirect/error flows.
10. `still-needed` Decide preset edit behavior for `trackable_template_id` clearing.
Reason: Preset ID is currently cleared only on name input edits in `web/static/trackable-presets.js:129`; edits to other preset-populated fields keep template linkage.
11. `still-needed` Reassess git-history artifact cleanup.
Reason: Artifacts are not currently tracked, but history still contains relevant paths (`git log -- server tmp/app data/app.db` returns commits including `1a9fe56` and `394760e`).
12. `still-needed` Add browser-level test for home quick-capture submit-state transitions.
Reason: No Playwright/browser test suite currently exists in the repository.
13. `still-needed` Add browser-level regression test for entries context actions.
Reason: No browser E2E tests currently validate context menu, dialog open, and delete-confirm DOM behavior.
14. `still-needed` Wrap `deleteAllUserData` in a DB transaction.
Reason: `internal/server/settings.go:261` performs sequential deletes without transactional rollback.
15. `still-needed` Add regression test for bottom-nav active state.
Reason: `internal/views/bottomNav.html` uses `aria-current="page"` switching, but no test asserts this mapping.

## Execution Order

1. Language correctness: items 1, 2, 3, 4, 5.
2. Data safety and backend consistency: items 7, 8, 14.
3. Product behavior decision: item 10.
4. Test coverage: items 9, 12, 13, 15.
5. Repo maintenance: item 11 (coordinate before history rewrite).

## Done Criteria

1. Every `still-needed` item is either implemented or converted to an explicit accepted risk with rationale.
2. `partial` items are closed by finishing remaining scope.
3. `not-needed` items are removed from `NEXT_STEPS.md` to keep backlog current.
4. Run `go test ./...` after backend/template changes.
