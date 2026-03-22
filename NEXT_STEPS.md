## Remaining Follow-ups

- Add structured diagnostics/metrics for WebAuthn ceremony failures.
	Why: `/webauthn/login/verify` currently returns a generic 400 for many error modes; logs now include origin/RPID context, but operators still need easy aggregation by failure reason.
	How: Add counters and structured logs for categories like missing ceremony cookie, origin mismatch, RP ID mismatch, sign-counter mismatch, and credential not found. Surface these in deployment dashboards/alerts.

- Coordinate git-history cleanup for prior runtime artifacts.
	Why: Artifacts are no longer tracked in current commits, but old history still contains paths such as `server`, `tmp/app`, and `data/app.db`.
	How: Align with collaborators, then run `git filter-repo --path server --path tmp/app --path data/app.db --invert-paths` and force-push with clear migration instructions.

- Add explicit approval step on the original device for QR link redemption.
	Why: QR-based linking is now implemented with short-lived, single-use, high-entropy tokens, but a photographed QR can still be redeemed by a third party before the user completes enrollment.
	How: Introduce a `pending` link status plus an approval prompt on the currently signed-in device. Allow `/auth/passkeys/register/options` on the new device only after approval, and expire pending links quickly.

- Remove the temporary legacy migration bridge after one release cycle.
	Why: `internal/server/migrations.go` now keeps the old custom migrator only to bootstrap existing installations into goose; once all active deployments have crossed that version boundary, this code is dead weight.
	How: Add an app version/date cutoff, then delete `runLegacyMigrations` and helpers once telemetry or release notes confirm all environments use `goose_db_version`.

- Make voice recorder bindings instance-safe (no global IDs).
	Why: `web/static/voice-recorder.js` currently binds via `getElementById`, which assumes one recorder on the page. Any future second recorder instance or partial swap with duplicate IDs can bind listeners to the wrong element.
	How: Replace IDs with `data-voice-*` hooks, initialize per `.voice-entry-section`, and scope queries/listeners to each section root.
