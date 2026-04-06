package server

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
	"roysland.me/symptomstracker/internal/db"
)

type propertyTB interface {
	Helper()
	Fatalf(string, ...any)
}

var imageSampleByMIME = map[string][]byte{
	"image/jpeg": {0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00},
	"image/png":  {0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
	"image/webp": {'R', 'I', 'F', 'F', 0x24, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '},
	"image/gif":  {'G', 'I', 'F', '8', '9', 'a'},
}

func uploadImageRequest(t propertyTB, s *Server, entryID int64, fileName string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", fileName)
	if err != nil {
		t.Fatalf("create image form part: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/entry/"+strconv.FormatInt(entryID, 10)+"/images", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	cookieResp := httptest.NewRecorder()
	if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
		t.Fatalf("set authenticated user cookie: %v", err)
	}
	for _, cookie := range cookieResp.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func insertEntryForProperty(t propertyTB, s *Server, userID int64) db.Entry {
	t.Helper()
	now := time.Now().UTC().Unix()
	entry, err := s.queries.CreateEntry(context.Background(), db.CreateEntryParams{
		UserID:                userID,
		RecordedAtUtc:         now,
		TimezoneOffsetMinutes: 0,
		EntryDate:             time.Now().Format(dateLayoutISO),
		NoteText:              sql.NullString{String: "property test entry", Valid: true},
		IsPrivate:             0,
		CreatedAtUtc:          now,
	})
	if err != nil {
		t.Fatalf("create entry fixture: %v", err)
	}
	return entry
}

// Feature: app-feature-roadmap, Property 7: Image size enforcement
func TestProp_ImageSizeEnforcement(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.ImageStorageDir = t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		entry := insertEntryForProperty(t, s, 1)
		sample := imageSampleByMIME["image/jpeg"]

		overLimit := rapid.Bool().Draw(t, "over_limit")
		size := rapid.IntRange(len(sample), maxImageUploadSize).Draw(t, "size")
		if overLimit {
			size = rapid.IntRange(maxImageUploadSize+1, maxImageUploadSize+1024).Draw(t, "size_over")
		}

		payload := make([]byte, size)
		copy(payload, sample)
		rr := uploadImageRequest(t, s, entry.ID, "size-test.jpg", payload)

		if overLimit {
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for over-limit upload (%d bytes), got %d", size, rr.Code)
			}
			return
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for in-limit upload (%d bytes), got %d", size, rr.Code)
		}
	})
}

// Feature: app-feature-roadmap, Property 8: Image MIME type enforcement
func TestProp_ImageMIMEEnforcement(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.ImageStorageDir = t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		entry := insertEntryForProperty(t, s, 1)

		isAllowed := rapid.Bool().Draw(t, "is_allowed")
		var payload []byte
		if isAllowed {
			mimeType := rapid.SampledFrom([]string{"image/jpeg", "image/png", "image/webp", "image/gif"}).Draw(t, "mime")
			payload = append([]byte(nil), imageSampleByMIME[mimeType]...)
			payload = append(payload, bytes.Repeat([]byte{0x00}, 32)...)
		} else {
			payload = []byte("this is plain text and should not be accepted as an image")
		}

		rr := uploadImageRequest(t, s, entry.ID, "mime-test.bin", payload)
		if isAllowed {
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for allowed MIME sample, got %d", rr.Code)
			}
			return
		}

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for disallowed MIME sample, got %d", rr.Code)
		}
	})
}

// Feature: app-feature-roadmap, Property 9: Image filename safety
func TestProp_ImageFilenameSafety(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.ImageStorageDir = t.TempDir()

	filePattern := regexp.MustCompile(`^1_[0-9]+\.(jpg|png|webp|gif)$`)

	rapid.Check(t, func(t *rapid.T) {
		entry := insertEntryForProperty(t, s, 1)

		untrustedFileName := rapid.StringMatching(`[a-zA-Z0-9._/-]{5,30}`).Draw(t, "untrusted_name")
		if !strings.Contains(untrustedFileName, "evil") {
			untrustedFileName += "-evil"
		}

		payload := append([]byte(nil), imageSampleByMIME["image/png"]...)
		payload = append(payload, bytes.Repeat([]byte{0x00}, 64)...)

		rr := uploadImageRequest(t, s, entry.ID, untrustedFileName, payload)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		rows, err := s.queries.GetImagesByEntryID(t.Context(), db.GetImagesByEntryIDParams{EntryID: entry.ID, UserID: 1})
		if err != nil {
			t.Fatalf("list images by entry: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected one image row, got %d", len(rows))
		}

		base := filepath.Base(rows[0].FilePath)
		if !filePattern.MatchString(base) {
			t.Fatalf("expected filename %q to match safe pattern", base)
		}
		if strings.Contains(base, "evil") || strings.Contains(base, filepath.Base(untrustedFileName)) {
			t.Fatalf("stored filename %q should not include user-supplied filename %q", base, untrustedFileName)
		}
	})
}

// Feature: app-feature-roadmap, Property 10: Image metadata persistence
func TestProp_ImageMetadataPersistence(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.ImageStorageDir = t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		entry := insertEntryForProperty(t, s, 1)

		mimeType := rapid.SampledFrom([]string{"image/jpeg", "image/png", "image/webp", "image/gif"}).Draw(t, "mime")
		payload := append([]byte(nil), imageSampleByMIME[mimeType]...)
		extraSize := rapid.IntRange(16, 256).Draw(t, "extra_size")
		payload = append(payload, bytes.Repeat([]byte{0x11}, extraSize)...)

		rr := uploadImageRequest(t, s, entry.ID, "persist-test.img", payload)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		rows, err := s.queries.GetImagesByEntryID(t.Context(), db.GetImagesByEntryIDParams{EntryID: entry.ID, UserID: 1})
		if err != nil {
			t.Fatalf("list images by entry: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected one metadata row, got %d", len(rows))
		}

		row := rows[0]
		if row.EntryID != entry.ID || row.UserID != 1 {
			t.Fatalf("unexpected metadata association: entry_id=%d user_id=%d", row.EntryID, row.UserID)
		}
		if row.FilePath == "" || row.MimeType == "" {
			t.Fatalf("expected non-empty file_path/mime_type, got path=%q mime=%q", row.FilePath, row.MimeType)
		}
		if row.OriginalSize != int64(len(payload)) {
			t.Fatalf("expected original_size=%d, got %d", len(payload), row.OriginalSize)
		}
	})
}

// Feature: app-feature-roadmap, Property 11: Image cascade delete
func TestProp_ImageCascadeDelete(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.ImageStorageDir = t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		entry := insertEntryForProperty(t, s, 1)

		imageCount := rapid.IntRange(1, 3).Draw(t, "image_count")
		for i := 0; i < imageCount; i++ {
			payload := append([]byte(nil), imageSampleByMIME["image/gif"]...)
			payload = append(payload, bytes.Repeat([]byte{byte(i + 1)}, 64)...)
			rr := uploadImageRequest(t, s, entry.ID, "cascade-test.gif", payload)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for upload %d, got %d", i, rr.Code)
			}
		}

		imagesBefore, err := s.queries.GetImagesByEntryID(t.Context(), db.GetImagesByEntryIDParams{EntryID: entry.ID, UserID: 1})
		if err != nil {
			t.Fatalf("list images before delete: %v", err)
		}
		if len(imagesBefore) != imageCount {
			t.Fatalf("expected %d images before delete, got %d", imageCount, len(imagesBefore))
		}

		req := httptest.NewRequest(http.MethodPost, "/entry/"+strconv.FormatInt(entry.ID, 10)+"/delete", nil)
		cookieResp := httptest.NewRecorder()
		if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
			t.Fatalf("set authenticated user cookie: %v", err)
		}
		for _, cookie := range cookieResp.Result().Cookies() {
			req.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 when deleting entry, got %d", rr.Code)
		}

		imagesAfter, err := s.queries.GetImagesByEntryID(t.Context(), db.GetImagesByEntryIDParams{EntryID: entry.ID, UserID: 1})
		if err != nil {
			t.Fatalf("list images after delete: %v", err)
		}
		if len(imagesAfter) != 0 {
			t.Fatalf("expected 0 images after delete, got %d", len(imagesAfter))
		}

		for _, image := range imagesBefore {
			if _, err := os.Stat(image.FilePath); !os.IsNotExist(err) {
				t.Fatalf("expected file %q to be removed, stat err=%v", image.FilePath, err)
			}
		}
	})
}
