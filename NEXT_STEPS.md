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

- Add authenticated device-link flow with QR for cross-device onboarding.
	Why: Discoverable passkeys and multi-passkey support are now in place, but users without synced passkey managers still need a smoother way to enroll a second device from an already signed-in device.
	How: Add a short-lived one-time linking token endpoint for authenticated users, render it as a QR code, and let the new device redeem it only to initiate `/auth/passkeys/options` for that same account before requiring passkey confirmation.
	Flow (end-to-end)
	1) On existing device (authenticated)

	User clicks: “Add new device”

	Server:

	Generate random token (high entropy, e.g. 32 bytes)

	Store:

	token
	user_id
	expires_at (e.g. 2–5 minutes)
	used = false

	Render QR
	2) On new device (QR scanned)

	User opens /link?t=TOKEN

	Server:

	Validate token:

	exists

	not expired

	not used

	If valid:

	Create temporary linking session (NOT a full auth session)

	Associate it with user_id

	Mark token as pending (optional)

	Now show:

	“Continue to add this device”

	3) Start WebAuthn registration (on new device)

	Call:

	/auth/passkeys/register/options

	But here’s the key:

	Instead of creating a new user,

	Use the user_id from the linking session

	So:

	user_id = linking_session.user_id
	4) Complete registration

	navigator.credentials.create()

	Send to server

	Verify

	Store credential under that user_id

	Now:

	Mark token as used

	Upgrade session → full authenticated session

	Important constraints (don’t skip these)
	1. Token must be:

	single-use

	short-lived (2–5 min)

	high entropy (unguessable)

	2. Token does NOT log the user in

	It only allows:

	starting registration for a specific user

	If you skip this distinction, you create an account takeover vector.

	3. Require user presence (WebAuthn)

	The new device must:

	perform biometric / PIN

	generate a real credential

	This is your actual security boundary.

	4. Optional but strong improvement

	Require confirmation on original device:

	Flow:

	New device scans QR

	Old device shows: “Approve new device?”

	Only then allow registration

	This prevents someone photographing the QR and hijacking it.
	UX notes (important for your app)

	Keep it very low friction:

	On existing device:

	“Add device”

	Show QR + short explanation

	On new device:

	Auto-continue after scan

	One button: “Use this device”

	Avoid extra steps unless necessary.
	Subtle edge case

	If user:

	scans QR

	waits too long

	→ token expires

	Handle gracefully:

	“Link expired, generate a new one”

