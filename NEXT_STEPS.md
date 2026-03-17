## Follow-ups after i18n key coverage cleanup

- Localize remaining hardcoded UI strings in `internal/views/topNav.html`.
	Why: The menu and theme toggle labels are still English-only and not routed through i18n.
	How: Replace hardcoded `aria-label` and button text with `t "..."` keys and add matching entries in `internal/i18n/i18n.go`.

- Localize client-side confirmation/error strings in `web/static/entries.js` and fallback message in `web/static/settings.js`.
	Why: Delete and confirmation flows still show hardcoded English browser dialogs.
	How: Pass translated strings through data attributes from templates and read those values in JavaScript instead of inline literals.
