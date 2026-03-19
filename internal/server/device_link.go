package server

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"roysland.me/symptomstracker/internal/db"
)

const (
	deviceLinkTokenTTL      = 5 * time.Minute
	deviceLinkSessionTTL    = 10 * time.Minute
	deviceLinkTokenByteSize = 32
)

func (s *Server) createDeviceLink(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	rawToken, err := generateDeviceLinkToken()
	if err != nil {
		respondInternalError(w, r, "Failed to create device link")
		return
	}

	hash := hashDeviceLinkToken(rawToken)
	now := time.Now().Unix()
	expiresAt := now + int64(deviceLinkTokenTTL/time.Second)
	if _, err := s.queries.CreateDeviceLinkToken(r.Context(), db.CreateDeviceLinkTokenParams{
		TokenHash:    hash,
		UserID:       userID,
		ExpiresAtUtc: expiresAt,
		CreatedAtUtc: now,
	}); err != nil {
		respondInternalError(w, r, "Failed to save device link")
		return
	}

	linkURL := absoluteURLForRequest(r, "/link?t="+url.QueryEscape(rawToken))
	qrPNG, err := qrcode.Encode(linkURL, qrcode.Medium, 256)
	if err != nil {
		respondInternalError(w, r, "Failed to generate QR code")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"link_url":       linkURL,
		"expires_at_utc": expiresAt,
		"qr_data_url":    "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG),
	})
}

func (s *Server) redeemDeviceLink(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	rawToken := strings.TrimSpace(r.URL.Query().Get("t"))
	if rawToken == "" {
		s.renderPage(w, r, "link_title", "link_content", map[string]any{"LinkReady": false})
		return
	}

	now := time.Now().Unix()
	token, err := s.queries.RedeemDeviceLinkToken(r.Context(), db.RedeemDeviceLinkTokenParams{
		TokenHash:     hashDeviceLinkToken(rawToken),
		NowUtc:        now,
		RedeemedAtUtc: sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			s.renderPage(w, r, "link_title", "link_content", map[string]any{"LinkReady": false})
			return
		}
		respondInternalError(w, r, "Failed to redeem device link")
		return
	}

	if err := s.authSessions.SetLinkingSession(w, token.UserID, token.ID, deviceLinkSessionTTL); err != nil {
		respondInternalError(w, r, "Failed to initialize linking session")
		return
	}

	s.renderPage(w, r, "link_title", "link_content", map[string]any{
		"LinkReady": true,
		"AutoStart": true,
	})
}

func absoluteURLForRequest(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		scheme = strings.TrimSpace(parts[0])
	}

	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
}

func generateDeviceLinkToken() (string, error) {
	buf := make([]byte, deviceLinkTokenByteSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashDeviceLinkToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
