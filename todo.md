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

