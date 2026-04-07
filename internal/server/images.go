package server

import (
	"bytes"
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

const maxImageUploadSize = 2 << 20 // 2 MB
const maxImageMultipartBodySize = maxImageUploadSize + (512 << 10)

var allowedImageMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (s *Server) uploadEntryImage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	entryID, ok := requirePathInt64(w, r, "id", "entry ID")
	if !ok {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if _, err := s.queries.GetEntryByID(r.Context(), db.GetEntryByIDParams{ID: entryID, UserID: userID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondNotFound(w, r, "Entry not found")
			return
		}
		respondInternalError(w, r, "Failed to fetch entry")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageMultipartBodySize)
	if err := r.ParseMultipartForm(maxImageMultipartBodySize); err != nil {
		respondBadRequest(w, r, "image upload exceeds 2 MB limit or is malformed")
		return
	}

	imageFile, _, err := r.FormFile("image")
	if err != nil {
		respondBadRequest(w, r, "image file is required")
		return
	}
	defer imageFile.Close()

	imageBytes, err := io.ReadAll(imageFile)
	if err != nil {
		respondBadRequest(w, r, "failed to read image file")
		return
	}
	if len(imageBytes) == 0 {
		respondBadRequest(w, r, "image file is empty")
		return
	}
	if len(imageBytes) > maxImageUploadSize {
		respondBadRequest(w, r, "image upload exceeds 2 MB limit")
		return
	}

	detected := http.DetectContentType(imageBytes)
	mimeType, _, err := mime.ParseMediaType(detected)
	if err != nil {
		mimeType = detected
	}
	ext, allowed := allowedImageMIMETypes[strings.ToLower(strings.TrimSpace(mimeType))]
	if !allowed {
		respondBadRequest(w, r, "unsupported image MIME type")
		return
	}

	if err := os.MkdirAll(s.cfg.ImageStorageDir, 0750); err != nil {
		log.Printf("failed to create image dir %s: %v", s.cfg.ImageStorageDir, err)
		respondInternalError(w, r, "failed to prepare image storage")
		return
	}
	root, err := os.OpenRoot(s.cfg.ImageStorageDir)
	if err != nil {
		log.Printf("failed to open image dir %s: %v", s.cfg.ImageStorageDir, err)
		respondInternalError(w, r, "failed to prepare image storage")
		return
	}
	defer root.Close()

	now := time.Now().UTC()
	fileName := fmt.Sprintf("%d_%d%s", userID, now.UnixNano(), ext)
	storedPath := filepath.ToSlash(filepath.Join(s.cfg.ImageStorageDir, fileName))

	dst, err := root.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Printf("failed to create image file %s: %v", storedPath, err)
		respondInternalError(w, r, "failed to save image")
		return
	}

	if _, err := io.Copy(dst, bytes.NewReader(imageBytes)); err != nil {
		_ = dst.Close()
		if removeErr := root.Remove(fileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove partial image file %s: %v", storedPath, removeErr)
		}
		respondInternalError(w, r, "failed to save image")
		return
	}

	if err := dst.Close(); err != nil {
		if removeErr := root.Remove(fileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove unreadable image file %s: %v", storedPath, removeErr)
		}
		respondInternalError(w, r, "failed to save image")
		return
	}

	imageRow, err := s.queries.InsertEntryImage(r.Context(), db.InsertEntryImageParams{
		EntryID:      entryID,
		UserID:       userID,
		FilePath:     storedPath,
		MimeType:     mimeType,
		OriginalSize: int64(len(imageBytes)),
		StorageTier:  "local",
		CreatedAtUtc: now.Unix(),
	})
	if err != nil {
		if removeErr := root.Remove(fileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("failed to remove image file after DB failure %s: %v", storedPath, removeErr)
		}
		respondInternalError(w, r, "failed to store image metadata")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<div class="image-upload-success" data-image-id="%d">Image uploaded</div>`, imageRow.ID)
}

func (s *Server) deleteEntryImage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	entryID, ok := requirePathInt64(w, r, "id", "entry ID")
	if !ok {
		return
	}
	imgID, ok := requirePathInt64(w, r, "imgID", "image ID")
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	imageRow, err := s.queries.GetImageByID(r.Context(), db.GetImageByIDParams{ID: imgID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondNotFound(w, r, "Image not found")
			return
		}
		respondInternalError(w, r, "Failed to fetch image")
		return
	}
	if imageRow.EntryID != entryID {
		respondNotFound(w, r, "Image not found")
		return
	}

	if err := s.removeImageFile(imageRow.FilePath); err != nil {
		log.Printf("warning: failed to delete image file %s: %v", imageRow.FilePath, err)
	}

	if err := s.queries.DeleteEntryImage(r.Context(), db.DeleteEntryImageParams{ID: imgID, UserID: userID}); err != nil {
		respondInternalError(w, r, "Failed to delete image metadata")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) removeImageFile(storedPath string) error {
	if storedPath == "" {
		return nil
	}

	root, err := os.OpenRoot(s.cfg.ImageStorageDir)
	if err != nil {
		return err
	}
	defer root.Close()

	name := filepath.Base(storedPath)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return fmt.Errorf("invalid image file path")
	}

	err = root.Remove(name)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
