## Remaining Follow-ups

- Add structured diagnostics/metrics for WebAuthn ceremony failures.
	Why: `/webauthn/login/verify` currently returns a generic 400 for many error modes; logs now include origin/RPID context, but operators still need easy aggregation by failure reason.
	How: Add counters and structured logs for categories like missing ceremony cookie, origin mismatch, RP ID mismatch, sign-counter mismatch, and credential not found. Surface these in deployment dashboards/alerts.

- Coordinate git-history cleanup for prior runtime artifacts.
	Why: Artifacts are no longer tracked in current commits, but old history still contains paths such as `server`, `tmp/app`, and `data/app.db`.
	How: Align with collaborators, then run `git filter-repo --path server --path tmp/app --path data/app.db --invert-paths` and force-push with clear migration instructions.

- Consider periodic polling / SSE for transcription status updates.
	Why: After a voice draft is saved the entry list shows "Transcribing..." indefinitely until the user manually refreshes. On real hardware whisper typically finishes in a few seconds.
	How: Either (a) add an HTMX polling target on the draft entry item (`hx-trigger="every 5s"` → `GET /entry/{id}`) that stops once `TranscriptionStatus` is no longer `pending`, or (b) push a Server-Sent Event when the worker finishes and let the client refresh the entry.

- Guard against duplicate voice-recorder event listeners after HTMX swaps.
	Why: `initVoiceRecorder` is called on every `htmx:afterSwap` and currently attaches new click listeners each time for the same DOM nodes, which can trigger multiple recorder/upload flows from one click.
	How: Make initialization idempotent by marking the section as bound (for example with `data-voice-bound="1"`) or by removing/replacing existing listeners before adding new ones.

- Add explicit approval step on the original device for QR link redemption.
	Why: QR-based linking is now implemented with short-lived, single-use, high-entropy tokens, but a photographed QR can still be redeemed by a third party before the user completes enrollment.
	How: Introduce a `pending` link status plus an approval prompt on the currently signed-in device. Allow `/auth/passkeys/register/options` on the new device only after approval, and expire pending links quickly.

- Persist the sensitive-content filter in user settings instead of browser-local storage.
	Why: The new entries-page toggle currently remembers its state only in the current browser, so a user switching devices or clearing storage gets inconsistent privacy behavior.
	How: Add a boolean setting such as `show_sensitive_content`, load it with the rest of `UserSettings`, and update the entries toggle to save through the existing settings flow while keeping the client-side instant hide/show behavior.

- Remove the temporary legacy migration bridge after one release cycle.
	Why: `internal/server/migrations.go` now keeps the old custom migrator only to bootstrap existing installations into goose; once all active deployments have crossed that version boundary, this code is dead weight.
	How: Add an app version/date cutoff, then delete `runLegacyMigrations` and helpers once telemetry or release notes confirm all environments use `goose_db_version`.
