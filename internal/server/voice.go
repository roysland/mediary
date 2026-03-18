package server

import (
	"database/sql"
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

	// Save the audio file under a name derived from user ID and time to avoid
	// collisions. We do NOT use any user-supplied filename.
	now := time.Now()
	audioFileName := fmt.Sprintf("%d_%d.webm", userID, now.UnixNano())
	audioFilePath := filepath.Join(audioDir, audioFileName)

	dst, err := os.OpenFile(audioFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		log.Printf("failed to create audio file %s: %v", audioFilePath, err)
		respondInternalError(w, r, "failed to save audio")
		return
	}
	if _, err := io.Copy(dst, audioFile); err != nil {
		dst.Close()
		os.Remove(audioFilePath)
		log.Printf("failed to write audio file %s: %v", audioFilePath, err)
		respondInternalError(w, r, "failed to save audio")
		return
	}
	dst.Close()

	// Create the draft entry in the database immediately.
	entry, err := s.queries.CreateDraftEntry(r.Context(), db.CreateDraftEntryParams{
		UserID:                userID,
		RecordedAtUtc:         now.UTC().Unix(),
		TimezoneOffsetMinutes: defaultTimezoneOffsetMinutes,
		EntryDate:             now.Format(dateLayoutISO),
		AudioFilePath:         sql.NullString{String: audioFilePath, Valid: true},
		CreatedAtUtc:          now.UTC().Unix(),
	})
	if err != nil {
		os.Remove(audioFilePath)
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
