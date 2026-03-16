# Agent Task 1 — Voice Logging (Speech-to-Text)

Priority: High
Objective: Enable users to quickly log symptoms without typing.

Functional Requirements
- Provide a large microphone button on the capture screen.
- Pressing the button transitions UI to Listening... mode.
- Audio recording stops automatically when the user presses stop or a duration limit is reached.
- Audio uploads automatically after recording.

UX Constraints
- Do not require transcription review immediately.
- Save the entry immediately as a draft.
- Transcription may appear later after processing.

Implementation Steps
Client: 
- Use the MediaRecorder API for capturing audio. 
- Limit recording duration (recommended: 30–60 seconds). 
- Upload audio using hx-post.

Server: 
- Store uploaded audio file. 
- Create the draft entry immediately.
- Return an immediate HTMX swap with a "Saved as Draft" confirmation. Do not keep the HTTP request open.
- Dispatch the transcription job to a background Go worker using whisper.cpp, entirely decoupled from the initial HTTP request.
- Update the entry with transcription once the background processing completes.

Failure Prevention
- Never block the UI waiting for transcription.
- Ensure the Go worker handles crashes or timeouts without bringing down the main web server.

------------------------------------------------------------------------

# Agent Task 2 — Transactional Draft System

Priority: High
Objective: Prevent loss of text entries due to crashes or session expiration.

Functional Requirements
- Autosave text input periodically while typing.
- Restore drafts when users return.
- Clear the draft once the entry is finalized.

Implementation Steps
Client: 
- Use HTMX autosave trigger: hx-trigger="keyup changed delay:2s"

Server: 
- Save drafts in SQLite.

Database structure:
drafts
------
id
user_id
draft_uuid
content
updated_at

Recovery Flow
When the user logs in:
1. Check for existing drafts.
2. Prompt: "You have an unsaved note from earlier. Restore it?"

Cleanup Phase
- When the user successfully submits the final entry, the server must delete the corresponding draft from the database. 

Failure Prevention
- Use Draft UUIDs to prevent multiple browser tabs from overwriting drafts.
- Avoid repeated writes if the content hash has not changed.
- Ensure orphaned drafts are cleaned up to prevent the database from filling up.

------------------------------------------------------------------------

# Agent Task 4 — Offline-First Logging (PWA)

Priority: High
Objective: Ensure logging works without network connectivity.

Functional Requirements
Entries must save locally if the server is unreachable.
User should immediately see a success state.

Implementation Steps
Client:
- Use IndexedDB to store pending entries.
- Implement a Service Worker queue to intercept failed network requests.

HTMX Specific Offline Handling:
- Because HTMX expects an HTML fragment, the Service Worker must intercept the failed hx-post request, queue the payload in IndexedDB, and immediately return a synthetic 200 OK response containing the "Saved locally ✓" HTML fragment. This prevents the UI from hanging or displaying an error.

Sync Process:
1. Save entry locally.
2. Return synthetic HTMX response to the user.
3. Attempt server sync in the background.
4. Retry automatically when connection returns.

Data Consistency
Each entry must include a client-generated timestamp (created_at). The server must trust this timestamp to ensure the timeline order remains correct when entries sync later.