package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"roysland.me/symptomstracker/internal/db"
)

const (
	shareTokenTTL      = 30 * time.Minute
	shareTokenByteSize = 20
	sharePasswordLen   = 7
)

var sharePasswordAlphabet = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

type shareScope struct {
	DateFrom string
	DateTo   string
	Private  bool
}

type shareTokenPreview struct {
	ID            int64
	ExpiresAtUTC  int64
	ScopeDateFrom string
	ScopeDateTo   string
	ScopePrivate  bool
}

func (s *Server) shareTokenRoute(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost, http.MethodDelete) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.shareTokenForm(w, r)
	case http.MethodPost:
		s.shareTokenSubmit(w, r)
	case http.MethodDelete:
		s.revokeShareToken(w, r)
	}
}

func (s *Server) createShareLink(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if !requireParsedForm(w, r) {
		return
	}

	scope := parseShareScope(r)
	if strings.TrimSpace(scope.DateTo) == "" {
		scope.DateTo = time.Now().UTC().Format(dateLayoutISO)
	}
	rawToken, tokenHash, err := generateShareToken()
	if err != nil {
		respondInternalError(w, r, "Failed to generate share token")
		return
	}

	password, err := generateSharePassword()
	if err != nil {
		respondInternalError(w, r, "Failed to generate share password")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		respondInternalError(w, r, "Failed to create share password")
		return
	}

	now := time.Now().UTC()
	createdAt := now.Unix()
	expiresAt := now.Add(shareTokenTTL).Unix()

	scopeFrom := sql.NullString{String: scope.DateFrom, Valid: strings.TrimSpace(scope.DateFrom) != ""}
	scopeTo := sql.NullString{String: scope.DateTo, Valid: strings.TrimSpace(scope.DateTo) != ""}
	scopePrivate := int64(0)
	if scope.Private {
		scopePrivate = 1
	}

	record, err := s.queries.CreateShareToken(r.Context(), db.CreateShareTokenParams{
		UserID:        userID,
		TokenHash:     tokenHash,
		PasswordHash:  passwordHash,
		ScopeDateFrom: scopeFrom,
		ScopeDateTo:   scopeTo,
		ScopePrivate:  scopePrivate,
		ExpiresAtUtc:  expiresAt,
		CreatedAtUtc:  createdAt,
	})
	if err != nil {
		respondInternalError(w, r, "Failed to save share token")
		return
	}

	tokenURL := absoluteURLForRequest(r, "/share/"+url.PathEscape(rawToken))
	qrPNG, err := qrcode.Encode(tokenURL, qrcode.Medium, 256)
	if err != nil {
		respondInternalError(w, r, "Failed to generate QR code")
		return
	}

	s.renderPage(w, r, "share_confirmation_title", "share_confirmation_content", map[string]any{
		"ShareURL":      tokenURL,
		"SharePassword": password,
		"ExpiresAtUTC":  expiresAt,
		"QRCodeDataURL": template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)),
		"Scope": map[string]any{
			"DateFrom": scope.DateFrom,
			"DateTo":   scope.DateTo,
			"Private":  scope.Private,
		},
		"TokenID": record.ID,
	})
}

func (s *Server) shareTokenForm(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	tokenValue := strings.TrimSpace(r.PathValue("token"))
	if tokenValue == "" {
		respondNotFound(w, r, "Share link not found")
		return
	}

	record, err := s.getActiveShareToken(r.Context(), tokenValue)
	if err != nil {
		if err == sql.ErrNoRows {
			respondNotFound(w, r, "Share link not found")
			return
		}
		respondInternalError(w, r, "Failed to load share link")
		return
	}

	s.renderPage(w, r, "share_password_title", "share_password_content", map[string]any{
		"Token":        tokenValue,
		"ExpiresAtUTC": record.ExpiresAtUtc,
	})
}

func (s *Server) shareTokenSubmit(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	tokenValue := strings.TrimSpace(r.PathValue("token"))
	if tokenValue == "" {
		respondNotFound(w, r, "Share link not found")
		return
	}

	if !requireParsedForm(w, r) {
		return
	}

	password := strings.TrimSpace(r.FormValue("password"))
	if password == "" {
		respondUnauthorized(w, r)
		return
	}

	tx, err := s.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalError(w, r, "Failed to process share link")
		return
	}

	qtx := s.queries.WithTx(tx)
	record, err := qtx.GetShareTokenByHash(r.Context(), hashShareToken(tokenValue))
	if err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			respondNotFound(w, r, "Share link not found")
			return
		}
		respondInternalError(w, r, "Failed to process share link")
		return
	}

	now := time.Now().UTC().Unix()
	if record.AccessedAtUtc.Valid || record.RevokedAtUtc.Valid || record.ExpiresAtUtc <= now {
		_ = tx.Rollback()
		respondNotFound(w, r, "Share link not found")
		return
	}

	if bcrypt.CompareHashAndPassword(record.PasswordHash, []byte(password)) != nil {
		_ = tx.Rollback()
		respondUnauthorized(w, r)
		return
	}

	if err := qtx.MarkShareTokenAccessed(r.Context(), db.MarkShareTokenAccessedParams{
		ID:            record.ID,
		AccessedAtUtc: sql.NullInt64{Int64: now, Valid: true},
	}); err != nil {
		_ = tx.Rollback()
		respondInternalError(w, r, "Failed to process share link")
		return
	}

	if err := tx.Commit(); err != nil {
		respondInternalError(w, r, "Failed to process share link")
		return
	}

	sharedEntries, err := s.buildSharedReportEntries(r.Context(), record)
	if err != nil {
		respondInternalError(w, r, "Failed to build shared report")
		return
	}

	s.renderPage(w, r, "share_report_title", "share_report_content", map[string]any{
		"ScopeDateFrom": nullStringValue(record.ScopeDateFrom),
		"ScopeDateTo":   nullStringValue(record.ScopeDateTo),
		"ScopePrivate":  record.ScopePrivate == 1,
		"AccessedAtUTC": now,
		"Entries":       sharedEntries,
	})
}

func (s *Server) revokeShareToken(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	tokenValue := strings.TrimSpace(r.PathValue("token"))
	if tokenValue == "" {
		respondNotFound(w, r, "Share link not found")
		return
	}

	record, err := s.queries.GetShareTokenByHash(r.Context(), hashShareToken(tokenValue))
	if err != nil {
		if err == sql.ErrNoRows {
			respondNotFound(w, r, "Share link not found")
			return
		}
		respondInternalError(w, r, "Failed to revoke share link")
		return
	}

	err = s.queries.RevokeShareToken(r.Context(), db.RevokeShareTokenParams{
		ID:           record.ID,
		UserID:       userID,
		RevokedAtUtc: sql.NullInt64{Int64: time.Now().UTC().Unix(), Valid: true},
	})
	if err != nil {
		respondInternalError(w, r, "Failed to revoke share link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listShareTokens(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	tokens, err := s.queries.ListActiveShareTokensByUser(r.Context(), db.ListActiveShareTokensByUserParams{
		UserID: userID,
		NowUtc: time.Now().UTC().Unix(),
	})
	if err != nil {
		respondInternalError(w, r, "Failed to list share links")
		return
	}

	items := make([]shareTokenPreview, 0, len(tokens))
	for _, token := range tokens {
		item := shareTokenPreview{
			ID:            token.ID,
			ExpiresAtUTC:  token.ExpiresAtUtc,
			ScopeDateFrom: nullStringValue(token.ScopeDateFrom),
			ScopeDateTo:   nullStringValue(token.ScopeDateTo),
			ScopePrivate:  token.ScopePrivate == 1,
		}
		items = append(items, item)
	}

	s.renderPage(w, r, "share_list_title", "share_list_content", map[string]any{
		"Tokens":        items,
		"CreateAction":  "/share/create",
		"DefaultToDate": time.Now().UTC().Format(dateLayoutISO),
	})
}

func (s *Server) buildSharedReportEntries(ctx context.Context, token db.ShareToken) ([]entryView, error) {
	rows, err := s.queries.ListEntries(ctx, db.ListEntriesParams{
		UserID:    token.UserID,
		EntryDate: "",
	})
	if err != nil {
		return nil, err
	}

	entries := buildEntryViews(rows)
	if len(entries) == 0 {
		return entries, nil
	}

	from := strings.TrimSpace(nullStringValue(token.ScopeDateFrom))
	to := strings.TrimSpace(nullStringValue(token.ScopeDateTo))
	includePrivate := token.ScopePrivate == 1

	filtered := make([]entryView, 0, len(entries))
	for _, entry := range entries {
		if entry.IsPrivate == 1 && !includePrivate {
			continue
		}
		if from != "" && entry.EntryDate < from {
			continue
		}
		if to != "" && entry.EntryDate > to {
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered, nil
}

func (s *Server) revokeShareTokenByID(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	tokenID, ok := requirePathInt64(w, r, "id", "share token id")
	if !ok {
		return
	}

	err := s.queries.RevokeShareToken(r.Context(), db.RevokeShareTokenParams{
		ID:           tokenID,
		UserID:       userID,
		RevokedAtUtc: sql.NullInt64{Int64: time.Now().UTC().Unix(), Valid: true},
	})
	if err != nil {
		respondInternalError(w, r, "Failed to revoke share link")
		return
	}

	http.Redirect(w, r, "/settings/shares", http.StatusSeeOther)
}

func parseShareScope(r *http.Request) shareScope {
	private := false
	rawPrivate := strings.TrimSpace(r.FormValue("scope_private"))
	if rawPrivate != "" {
		parsedBool, err := strconv.ParseBool(rawPrivate)
		if err == nil {
			private = parsedBool
		} else if rawPrivate == "1" || strings.EqualFold(rawPrivate, "on") {
			private = true
		}
	}

	return shareScope{
		DateFrom: strings.TrimSpace(r.FormValue("scope_date_from")),
		DateTo:   strings.TrimSpace(r.FormValue("scope_date_to")),
		Private:  private,
	}
}

func generateShareToken() (rawToken string, tokenHash []byte, err error) {
	buf := make([]byte, shareTokenByteSize)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}

	rawToken = base64.RawURLEncoding.EncodeToString(buf)
	return rawToken, hashShareToken(rawToken), nil
}

func generateSharePassword() (string, error) {
	buf := make([]byte, sharePasswordLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	out := make([]byte, sharePasswordLen)
	for i := range buf {
		out[i] = sharePasswordAlphabet[int(buf[i])%len(sharePasswordAlphabet)]
	}

	return string(out), nil
}

func hashShareToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func (s *Server) getActiveShareToken(ctx context.Context, rawToken string) (db.ShareToken, error) {
	record, err := s.queries.GetShareTokenByHash(ctx, hashShareToken(rawToken))
	if err != nil {
		return db.ShareToken{}, err
	}

	now := time.Now().UTC().Unix()
	if record.AccessedAtUtc.Valid || record.RevokedAtUtc.Valid || record.ExpiresAtUtc <= now {
		return db.ShareToken{}, sql.ErrNoRows
	}

	return record, nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
