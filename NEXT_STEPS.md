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

- Install whisper.cpp to enable voice transcription.
	Why: The transcription worker is fully implemented but needs the whisper.cpp binary and a ggml model to produce text. Without it, voice drafts are saved but transcription_status is set to 'failed' and no text appears.
	How:
	  1. Clone and build whisper.cpp: `git clone https://github.com/ggerganov/whisper.cpp && cd whisper.cpp && make`
	  2. Download a model: `bash models/download-ggml-model.sh base.en` (or `small` for better accuracy)
	  3. Set env vars before running the app:
	     ```
	     WHISPER_BINARY_PATH=/path/to/whisper.cpp/main
	     WHISPER_MODEL_PATH=/path/to/whisper.cpp/models/ggml-base.en.bin
	     FFMPEG_BINARY_PATH=ffmpeg  # or full path; ffmpeg must be installed
	     ```
	  4. ffmpeg is required to convert browser WebM audio to 16kHz WAV: `dnf install ffmpeg` or `apt install ffmpeg`.

- Add CSS styles for the voice recording UI.
	Why: The voice section elements (#voice-idle, #voice-recording, .btn-voice-mic, .voice-draft-badge, .voice-saved, etc.) have no dedicated styles yet and rely on fallback defaults.
	How: Add the required rules to EXTERNAL_DEPS.md for the stylesheet maintainer, or extend web/static/style.css. Key elements to style: `.voice-idle`, `.btn-voice-mic` (large circular button), `.voice-recording`, `.voice-dot` (animated pulsing indicator), `.voice-draft-badge`, `.voice-saved`, `.voice-error`.

- Consider periodic polling / SSE for transcription status updates.
	Why: After a voice draft is saved the entry list shows "Transcribing..." indefinitely until the user manually refreshes. On real hardware whisper typically finishes in a few seconds.
	How: Either (a) add an HTMX polling target on the draft entry item (`hx-trigger="every 5s"` → `GET /entry/{id}`) that stops once `TranscriptionStatus` is no longer `pending`, or (b) push a Server-Sent Event when the worker finishes and let the client refresh the entry.

