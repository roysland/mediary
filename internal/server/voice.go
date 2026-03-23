package server

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

const (
	maxAudioUploadSize  = 10 << 20 // 10 MB
	maxRecordingSeconds = 60
)

func (s *Server) addVoiceEntry(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAudioUploadSize)
	if err := r.ParseMultipartForm(maxAudioUploadSize); err != nil {
		respondBadRequest(w, r, "audio upload too large or malformed")
		return
	}

	audioFile, header, err := r.FormFile("audio")
	if err != nil {
		respondBadRequest(w, r, "audio file is required")
		return
	}
	defer audioFile.Close()

	// Only allow audio content-types or octet-stream (browser sends different types).
	contentType := header.Header.Get("Content-Type")
	if !isAllowedAudioContentType(contentType) {
		respondBadRequest(w, r, "unsupported audio format")
		return
	}

	// Ensure storage directory exists.
	audioDir := s.cfg.AudioStorageDir
	if err := os.MkdirAll(audioDir, 0750); err != nil {
		log.Printf("failed to create audio dir %s: %v", audioDir, err)
		respondInternalError(w, r, "failed to prepare storage")
		return
	}
	root, err := os.OpenRoot(audioDir)
	if err != nil {
		log.Printf("failed to open audio dir %s: %v", audioDir, err)
		respondInternalError(w, r, "failed to prepare storage")
		return
	}
	defer root.Close()

	// Save the audio file under a name derived from user ID and time to avoid
	// collisions. We do NOT use any user-supplied filename.
	now := time.Now()
	audioFileName := fmt.Sprintf("%d_%d.webm", userID, now.UnixNano())
	audioFilePath := filepath.Join(audioDir, audioFileName)
	// Store only the relative web path in the database, not the full filesystem path.
	// This ensures it works correctly regardless of where AudioStorageDir is located.
	relativeAudioPath := filepath.ToSlash(filepath.Join("data/audio", audioFileName))

	dst, err := root.OpenFile(audioFileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Printf("failed to create audio file %s: %v", audioFilePath, err)
		respondInternalError(w, r, "failed to save audio")
		return
	}
	if _, err := io.Copy(dst, audioFile); err != nil {
		if closeErr := dst.Close(); closeErr != nil {
			log.Printf("failed to close audio file %s: %v", audioFilePath, closeErr)
		}
		if removeErr := root.Remove(audioFileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove partial audio file %s: %v", audioFilePath, removeErr)
		}
		log.Printf("failed to write audio file %s: %v", audioFilePath, err)
		respondInternalError(w, r, "failed to save audio")
		return
	}
	if err := dst.Close(); err != nil {
		if removeErr := root.Remove(audioFileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove unreadable audio file %s: %v", audioFilePath, removeErr)
		}
		log.Printf("failed to finalize audio file %s: %v", audioFilePath, err)
		respondInternalError(w, r, "failed to save audio")
		return
	}

	// Create the draft entry in the database immediately.
	entry, err := s.queries.CreateDraftEntry(r.Context(), db.CreateDraftEntryParams{
		UserID:                userID,
		RecordedAtUtc:         now.UTC().Unix(),
		TimezoneOffsetMinutes: defaultTimezoneOffsetMinutes,
		EntryDate:             now.Format(dateLayoutISO),
		AudioFilePath:         sql.NullString{String: relativeAudioPath, Valid: true},
		CreatedAtUtc:          now.UTC().Unix(),
	})
	if err != nil {
		if removeErr := root.Remove(audioFileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove orphaned audio file %s: %v", audioFilePath, removeErr)
		}
		log.Printf("failed to create draft entry: %v", err)
		respondInternalError(w, r, "failed to save entry")
		return
	}

	// Dispatch the transcription job asynchronously — handler returns immediately.
	s.transcriptionWorker.Enqueue(r.Context(), TranscriptionJob{
		EntryID:       entry.ID,
		AudioFilePath: audioFilePath,
	})

	// Return an HTMX-ready confirmation fragment.
	s.renderTemplate(w, r, "voice_saved_content", map[string]interface{}{
		"EntryID":   entry.ID,
		"EntryDate": entry.EntryDate,
	})
}

// isAllowedAudioContentType returns true for content-types that browsers send
// when recording via MediaRecorder. The list is intentionally restrictive.
func isAllowedAudioContentType(ct string) bool {
	baseType := strings.TrimSpace(strings.ToLower(ct))
	if baseType != "" {
		parsed, _, err := mime.ParseMediaType(baseType)
		if err == nil {
			baseType = parsed
		}
	}

	allowed := []string{
		"audio/webm",
		"audio/ogg",
		"audio/mp4",
		"audio/mpeg",
		"audio/wav",
		"application/octet-stream", // Chrome sometimes sends this
		"",                         // some browsers omit the header
	}
	for _, a := range allowed {
		if baseType == a {
			return true
		}
	}
	return false
}
