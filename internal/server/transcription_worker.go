package server

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

// TranscriptionJob represents a pending audio transcription task.
type TranscriptionJob struct {
	EntryID       int64
	AudioFilePath string
}

// TranscriptionWorker processes audio files in the background using whisper.cpp.
type TranscriptionWorker struct {
	jobs              chan TranscriptionJob
	queries           *db.Queries
	whisperBinaryPath string
	whisperModelPath  string
	ffmpegBinaryPath  string
	timeoutSeconds    int
}

func newTranscriptionWorker(queries *db.Queries, cfg Config) *TranscriptionWorker {
	return &TranscriptionWorker{
		jobs:              make(chan TranscriptionJob, 20),
		queries:           queries,
		whisperBinaryPath: cfg.WhisperBinaryPath,
		whisperModelPath:  cfg.WhisperModelPath,
		ffmpegBinaryPath:  cfg.FFmpegBinaryPath,
		timeoutSeconds:    cfg.TranscriptionTimeoutSeconds,
	}
}

// Start launches the background worker goroutine. It recovers from panics
// and continues processing until ctx is cancelled.
func (w *TranscriptionWorker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-w.jobs:
				if !ok {
					return
				}
				w.safeProcess(ctx, job)
			}
		}
	}()
}

// Enqueue adds a job to the queue without blocking. If the queue is full the
// entry is immediately marked as failed so the user can see something went wrong.
func (w *TranscriptionWorker) Enqueue(ctx context.Context, job TranscriptionJob) {
	select {
	case w.jobs <- job:
	default:
		log.Printf("transcription queue full; marking entry %d as failed", job.EntryID)
		if err := w.queries.MarkTranscriptionFailed(ctx, job.EntryID); err != nil {
			log.Printf("failed to mark transcription failed for entry %d: %v", job.EntryID, err)
		}
	}
}

// RecoverPending re-enqueues entries whose transcription was interrupted (e.g.
// by a server restart). Should be called once at startup after the worker is
// started.
func (w *TranscriptionWorker) RecoverPending(ctx context.Context) {
	rows, err := w.queries.ListPendingTranscriptions(ctx)
	if err != nil {
		log.Printf("transcription worker: failed to load pending transcriptions: %v", err)
		return
	}
	for _, row := range rows {
		if !row.AudioFilePath.Valid || row.AudioFilePath.String == "" {
			continue
		}
		log.Printf("transcription worker: re-queuing pending entry %d", row.ID)
		w.Enqueue(ctx, TranscriptionJob{
			EntryID:       row.ID,
			AudioFilePath: row.AudioFilePath.String,
		})
	}
}

func (w *TranscriptionWorker) safeProcess(ctx context.Context, job TranscriptionJob) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("transcription worker: panic for entry %d: %v", job.EntryID, r)
			_ = w.queries.MarkTranscriptionFailed(ctx, job.EntryID)
		}
	}()
	w.processJob(ctx, job)
}

func (w *TranscriptionWorker) processJob(ctx context.Context, job TranscriptionJob) {
	if w.whisperBinaryPath == "" || w.whisperModelPath == "" {
		log.Printf("transcription worker: whisper not configured; skipping entry %d", job.EntryID)
		if err := w.queries.MarkTranscriptionFailed(ctx, job.EntryID); err != nil {
			log.Printf("failed to mark transcription failed for entry %d: %v", job.EntryID, err)
		}
		return
	}

	// Verify the audio file exists before starting.
	if _, err := os.Stat(job.AudioFilePath); err != nil {
		log.Printf("transcription worker: audio file not found for entry %d: %v", job.EntryID, err)
		if err2 := w.queries.MarkTranscriptionFailed(ctx, job.EntryID); err2 != nil {
			log.Printf("failed to mark transcription failed for entry %d: %v", job.EntryID, err2)
		}
		return
	}

	timeout := time.Duration(w.timeoutSeconds) * time.Second
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Convert the uploaded audio (webm/opus) to 16kHz mono WAV that whisper expects.
	wavPath, err := w.convertToWAV(jobCtx, job.AudioFilePath)
	if err != nil {
		log.Printf("transcription worker: ffmpeg conversion failed for entry %d: %v", job.EntryID, err)
		if err2 := w.queries.MarkTranscriptionFailed(ctx, job.EntryID); err2 != nil {
			log.Printf("failed to mark transcription failed for entry %d: %v", job.EntryID, err2)
		}
		return
	}
	defer os.Remove(wavPath)

	text, err := w.runWhisper(jobCtx, wavPath)
	if err != nil {
		log.Printf("transcription worker: whisper failed for entry %d: %v", job.EntryID, err)
		if err2 := w.queries.MarkTranscriptionFailed(ctx, job.EntryID); err2 != nil {
			log.Printf("failed to mark transcription failed for entry %d: %v", job.EntryID, err2)
		}
		return
	}

	text = strings.TrimSpace(text)
	if err := w.queries.UpdateEntryTranscription(ctx, db.UpdateEntryTranscriptionParams{
		NoteText: sql.NullString{String: text, Valid: text != ""},
		ID:       job.EntryID,
	}); err != nil {
		log.Printf("transcription worker: failed to update entry %d with transcription: %v", job.EntryID, err)
		return
	}

	log.Printf("transcription worker: completed entry %d (%d chars)", job.EntryID, len(text))
}

// convertToWAV converts any audio format to 16kHz mono WAV using ffmpeg.
// Returns the path to the temporary WAV file (caller must remove it).
func (w *TranscriptionWorker) convertToWAV(ctx context.Context, inputPath string) (string, error) {
	wavPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "_converted.wav"

	// #nosec G204 — ffmpegBinaryPath is loaded from operator-controlled env config, not user input.
	cmd := exec.CommandContext(ctx, w.ffmpegBinaryPath,
		"-i", inputPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-y",
		wavPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", &transcriptionError{msg: "ffmpeg", detail: string(out), cause: err}
	}
	return wavPath, nil
}

// runWhisper calls whisper.cpp CLI and returns the transcribed text from stdout.
func (w *TranscriptionWorker) runWhisper(ctx context.Context, wavPath string) (string, error) {
	// #nosec G204 — whisperBinaryPath is loaded from operator-controlled env config, not user input.
	cmd := exec.CommandContext(ctx, w.whisperBinaryPath,
		"-m", w.whisperModelPath,
		"-f", wavPath,
		"-nt", // no timestamps
		"-np", // no progress
		"--no-prints",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", &transcriptionError{msg: "whisper", detail: string(out), cause: err}
	}
	return string(out), nil
}

type transcriptionError struct {
	msg    string
	detail string
	cause  error
}

func (e *transcriptionError) Error() string {
	return e.msg + ": " + e.cause.Error() + " — " + e.detail
}
