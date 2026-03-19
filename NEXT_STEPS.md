## Remaining Follow-ups

- Coordinate git-history cleanup for prior runtime artifacts.
	Why: Artifacts are no longer tracked in current commits, but old history still contains paths such as `server`, `tmp/app`, and `data/app.db`.
	How: Align with collaborators, then run `git filter-repo --path server --path tmp/app --path data/app.db --invert-paths` and force-push with clear migration instructions.

- Add browser-level test for home quick-capture submit-state transitions.
	Why: Server-side tests cover hooks, but they do not verify client-side loading/success/idle state transitions.
	How: Add a Playwright test that submits the home quick-capture form and asserts submit-state attribute transitions and input reset behavior.

- Add browser-level regression test for entries context actions.
	Why: Entries context menu, add-trackable dialog opening, and delete confirmation are browser-driven and can regress without server test failures.
	How: Add a Playwright test that opens the entry context menu, verifies dialog behavior, and confirms deletion updates the DOM only after successful confirmation.

- Add CSS styles for the voice recording UI.
	Why: The voice section elements (#voice-idle, #voice-recording, .btn-voice-mic, .voice-draft-badge, .voice-saved, etc.) have no dedicated styles yet and rely on fallback defaults.
	How: Add the required rules to EXTERNAL_DEPS.md for the stylesheet maintainer, or extend web/static/style.css. Key elements to style: `.voice-idle`, `.btn-voice-mic` (large circular button), `.voice-recording`, `.voice-dot` (animated pulsing indicator), `.voice-draft-badge`, `.voice-saved`, `.voice-error`.

- Consider periodic polling / SSE for transcription status updates.
	Why: After a voice draft is saved the entry list shows "Transcribing..." indefinitely until the user manually refreshes. On real hardware whisper typically finishes in a few seconds.
	How: Either (a) add an HTMX polling target on the draft entry item (`hx-trigger="every 5s"` → `GET /entry/{id}`) that stops once `TranscriptionStatus` is no longer `pending`, or (b) push a Server-Sent Event when the worker finishes and let the client refresh the entry.

- Guard against duplicate voice-recorder event listeners after HTMX swaps.
	Why: `initVoiceRecorder` is called on every `htmx:afterSwap` and currently attaches new click listeners each time for the same DOM nodes, which can trigger multiple recorder/upload flows from one click.
	How: Make initialization idempotent by marking the section as bound (for example with `data-voice-bound="1"`) or by removing/replacing existing listeners before adding new ones.

- Verify `CrossOriginProtection` behavior behind reverse proxies and multi-host deployments.
	Why: CSRF checks depend on request host/origin matching. If TLS termination or host rewriting is misconfigured, legitimate same-origin writes may be rejected or protection may not match intended public origins.
	How: In staging/prod, validate `Origin` and `Host` handling end-to-end and add explicit trusted origins via `AddTrustedOrigin` if traffic comes through alternate public domains.
