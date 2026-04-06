# Symptom Tracker

A private, low-friction diary and symptom tracker focused on people living with ME/CFS and similar chronic conditions.

The app is mobile-first and optimized for low cognitive load: quick text logging, voice capture, privacy controls, and lightweight navigation.

## Why this app exists

The product is built around a few constraints:

- Logging should take as little energy as possible.
- Privacy should be the default, not an add-on.
- UI should be touch-friendly and predictable when users are fatigued.
- Save feedback must be explicit and trustworthy.

## Core user experience

The app uses a consistent shell with bottom navigation:

- Home
- Entries
- Trackables
- Settings

Main flows:

1. Authenticate with passkeys (WebAuthn), no passwords or email.
2. Capture notes quickly on Home, with optional private mode.
3. Record voice notes with clear recording/upload/success/error states.
4. Browse diary entries by day with trackable values and contextual actions.
5. Define and log custom trackables (integer, boolean, text).
6. Manage preferences, passkeys, exports, and destructive data actions in Settings.

## Feature overview

### Authentication

- Passwordless sign-in and registration via WebAuthn passkeys.
- Optional device name during registration.
- Dedicated cross-device linking flow using short-lived QR links.

### Home quick capture

- Text entry card with private toggle.
- Voice entry card with explicit states:
	- idle
	- recording (timer + stop)
	- uploading
	- success
	- error
- Universal submit animation pattern: idle -> loading spinner -> success check.

### Entries

- Day-based diary browsing with a 7-day strip.
- Entry cards with timestamp, note/transcription status, private markers, and attached trackables.
- Context menu actions: edit text, add trackable, delete.
- Swipe-to-delete interaction.
- Sensitive content filter persisted in browser storage.

### Trackables

- User-defined metrics with three value types:
	- integer (range)
	- boolean (yes/no)
	- text
- Preset templates with autofill behavior.
- Sensitive trackables with optional private label.
- Daily dismiss/restore behavior for trackable rows.

### Settings

- Preferences: language, theme, screen lock preference, share timer.
- Security actions: add passkey, add another device, log out.
- Data actions: export all data, clear all data with double confirmation.

### Voice transcription

- Voice upload creates draft entries immediately.
- Background worker runs speech-to-text (Whisper) and updates entries when complete.
- Draft states include pending and failed.

### Internationalization

- English and Norwegian supported.
- Translation-key based rendering in templates.

## Routes

- `/auth` - login/register with passkey
- `/` - Home quick capture
- `/entries` - diary browsing
- `/trackables` - trackable picker/list
- `/trackables/add` - add trackable
- `/settings` - preferences and account/data actions
- `/link?t=<token>` - cross-device passkey enrollment

## Tech stack

- Go server-rendered web app
- SQLite storage
- WebAuthn for passwordless auth
- HTML templates + static JavaScript/CSS
- HTMX/AJAX style partial updates in key flows
- sqlc for typed query generation
- goose migrations
- Playwright browser tests

## Project structure

- `cmd/server/` - application entrypoint
- `internal/server/` - routing, handlers, services, template wiring
- `internal/views/` - HTML templates/partials
- `web/static/` - browser JavaScript and styles
- `db/` - schema, SQL queries, migrations
- `internal/db/` - sqlc generated database code (do not edit manually)
- `tests/browser/` - Playwright specs
- `docs/` - operational and security notes

## Local development

### Prerequisites

- Go 1.25+
- C toolchain for `github.com/mattn/go-sqlite3` (CGO)
- Node.js (for browser tests)
- Optional for voice transcription:
	- `whisper.cpp`
	- ggml model
	- `ffmpeg`

### Run the app

```bash
go run ./cmd/server/main.go
```

Server defaults include:

- listen address: `:8080`
- sqlite path: `data/app.db`
- app environment: `development`

### Build

```bash
go build ./cmd/server
```

### Test

```bash
go test ./...
```

Browser tests:

```bash
npm install
npm run test:browser
```

Live-server Playwright project:

```bash
npm run test:browser:live
```

## Configuration

The app reads configuration from environment variables.

Common variables:

- `APP_ENV` (default: `development`)
- `DB_PATH` (default: `data/app.db`)
- `LISTEN_ADDR` (default: `:8080`)
- `AUTH_SESSION_SECRET` (required in production)
- `WEBAUTHN_RP_ID` (default: `localhost`)
- `WEBAUTHN_RP_DISPLAY_NAME` (default: `Symptoms Tracker`)
- `WEBAUTHN_RP_ORIGINS` (default: `http://localhost:8080`)
- `CSRF_TRUSTED_ORIGINS` (comma-separated)

Voice/transcription variables:

- `AUDIO_STORAGE_DIR` (default: `data/audio`)
- `WHISPER_BINARY_PATH` (empty disables transcription)
- `WHISPER_MODEL_PATH`
- `FFMPEG_BINARY_PATH` (default: `ffmpeg`)
- `TRANSCRIPTION_TIMEOUT_SECONDS` (default: `120`)

E2E helper variable:

- `PLAYWRIGHT_E2E_AUTH_TOKEN` (test-only auth bypass)

In non-production environments, `.env` is loaded automatically if present.

## Voice transcription setup

To prepare transcription dependencies:

```bash
./scripts/setup-transcription.sh
```

See `docs/transcription.md` for the required runtime tools.

## Database and SQL workflow

- Migrations run at startup.
- SQL schema/query source of truth lives in `db/schema.sql` and `db/queries.sql`.
- sqlc generated files live in `internal/db/`.

Regenerate sqlc code after SQL changes:

```bash
sqlc generate
```

Important:

- Do not manually edit files in `internal/db/`.

## Security checks

```bash
npm run security:gosec
```

Write report to file:

```bash
npm run security:gosec:report
```

## Design and UX principles

- Mobile-first and thumb-friendly interactions.
- Clear, explicit save feedback on forms.
- Privacy controls should be visible and easy to understand.
- Low-clutter layouts to reduce cognitive load.
- Light and dark themes are both first-class.

## Contributor notes

- Keep server handlers/routing in `internal/server/`.
- Route template rendering through existing rendering helpers.
- For HTMX/AJAX interactions, return successful server responses before removing/replacing DOM content.
- If database schema or query logic changes, update SQL files then regenerate sqlc output.

## License

No license has been declared in this repository yet.
