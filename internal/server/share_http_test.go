package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"roysland.me/symptomstracker/internal/db"
)

func TestShareCreateFlowPersistsTokenAndRendersConfirmation(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("scope_date_from=2026-01-01&scope_date_to=2026-01-31&scope_private=1")
	req := authedRequest(t, s, http.MethodPost, "/share/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	responseBody := rr.Body.String()
	if !strings.Contains(responseBody, "Share link created") {
		t.Fatalf("expected share confirmation content")
	}
	if !strings.Contains(responseBody, "/share/") {
		t.Fatalf("expected response to include share URL")
	}

	tokens, err := s.queries.ListActiveShareTokensByUser(context.Background(), db.ListActiveShareTokensByUserParams{
		UserID: 1,
		NowUtc: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("list active share tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one active token, got %d", len(tokens))
	}
	if tokens[0].ScopePrivate != 1 {
		t.Fatalf("expected scope_private=1, got %d", tokens[0].ScopePrivate)
	}
	if !tokens[0].ScopeDateFrom.Valid || tokens[0].ScopeDateFrom.String != "2026-01-01" {
		t.Fatalf("unexpected scope_date_from: %#v", tokens[0].ScopeDateFrom)
	}
	if !tokens[0].ScopeDateTo.Valid || tokens[0].ScopeDateTo.String != "2026-01-31" {
		t.Fatalf("unexpected scope_date_to: %#v", tokens[0].ScopeDateTo)
	}
}

func TestShareTokenIsSingleUse(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rawToken := "share-single-use-token"
	password := "ABC2345"
	seedShareToken(t, s, rawToken, password, time.Now().UTC().Add(5*time.Minute).Unix())

	firstReq := httptest.NewRequest(http.MethodPost, "/share/"+rawToken, strings.NewReader("password="+password))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstRR := httptest.NewRecorder()
	s.ServeHTTP(firstRR, firstReq)

	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first use to return 200, got %d: %s", firstRR.Code, firstRR.Body.String())
	}
	if !strings.Contains(firstRR.Body.String(), "Shared report") {
		t.Fatalf("expected shared report content on first use")
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/share/"+rawToken, strings.NewReader("password="+password))
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondRR := httptest.NewRecorder()
	s.ServeHTTP(secondRR, secondReq)

	if secondRR.Code != http.StatusNotFound {
		t.Fatalf("expected second use to return 404, got %d", secondRR.Code)
	}
}

func TestShareTokenExpiredReturnsNotFound(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rawToken := "share-expired-token"
	seedShareToken(t, s, rawToken, "DEF2345", time.Now().UTC().Add(-1*time.Minute).Unix())

	req := httptest.NewRequest(http.MethodGet, "/share/"+rawToken, nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for expired token, got %d", rr.Code)
	}
}

func TestShareTokenRevocationBlocksAccess(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rawToken := "share-revoke-token"
	token := seedShareToken(t, s, rawToken, "GHI2345", time.Now().UTC().Add(10*time.Minute).Unix())

	revokeReq := authedRequest(t, s, http.MethodPost, "/settings/shares/"+int64ToString(token.ID)+"/revoke", nil)
	revokeRR := httptest.NewRecorder()
	s.ServeHTTP(revokeRR, revokeReq)

	if revokeRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on revoke route, got %d", revokeRR.Code)
	}
	if location := revokeRR.Header().Get("Location"); location != "/settings/shares" {
		t.Fatalf("expected redirect to /settings/shares, got %q", location)
	}

	shareReq := httptest.NewRequest(http.MethodGet, "/share/"+rawToken, nil)
	shareRR := httptest.NewRecorder()
	s.ServeHTTP(shareRR, shareReq)

	if shareRR.Code != http.StatusNotFound {
		t.Fatalf("expected revoked token to return 404, got %d", shareRR.Code)
	}
}

func TestShareRoutesIncludeSecurityHeaders(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rawToken := "share-header-token"
	password := "JKL2345"
	seedShareToken(t, s, rawToken, password, time.Now().UTC().Add(5*time.Minute).Unix())

	getReq := httptest.NewRequest(http.MethodGet, "/share/"+rawToken, nil)
	getRR := httptest.NewRecorder()
	s.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200 on share token form, got %d", getRR.Code)
	}
	assertShareSecurityHeaders(t, getRR)

	postReq := httptest.NewRequest(http.MethodPost, "/share/"+rawToken, strings.NewReader("password=WRONG"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRR := httptest.NewRecorder()
	s.ServeHTTP(postRR, postReq)

	if postRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on wrong password, got %d", postRR.Code)
	}
	assertShareSecurityHeaders(t, postRR)
}

func TestShareCreateDefaultsToDateToToday(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	body := strings.NewReader("scope_date_from=2026-01-01")
	req := authedRequest(t, s, http.MethodPost, "/share/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	tokens, err := s.queries.ListActiveShareTokensByUser(context.Background(), db.ListActiveShareTokensByUserParams{
		UserID: 1,
		NowUtc: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("list active share tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one active token, got %d", len(tokens))
	}

	want := time.Now().UTC().Format(dateLayoutISO)
	if !tokens[0].ScopeDateTo.Valid || tokens[0].ScopeDateTo.String != want {
		t.Fatalf("expected scope_date_to=%q, got %#v", want, tokens[0].ScopeDateTo)
	}
}

func TestShareReportIncludesMatchingEntries(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	entryInRange := insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: "2026-01-10", note: "included note", recordedAtUTC: 1710000000})
	insertEntryFixture(t, s, entryFixture{userID: 1, entryDate: "2026-02-15", note: "excluded note", recordedAtUTC: 1711000000})

	trackable := insertTrackableDefinitionFixture(t, s, trackableDefinitionFixture{
		userID:      1,
		name:        "Fatigue",
		valueType:   "integer",
		icon:        "⚡",
		category:    "symptom",
		isSensitive: 0,
	})

	_, err := s.queries.CreateTrackableValue(context.Background(), db.CreateTrackableValueParams{
		EntryID:               entryInRange.ID,
		TrackableDefinitionID: trackable.ID,
		ValueInt:              sql.NullInt64{Int64: 7, Valid: true},
		CreatedAtUtc:          time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("seed trackable value: %v", err)
	}

	rawToken := "share-report-token"
	password := "MNO2345"
	token := seedShareToken(t, s, rawToken, password, time.Now().UTC().Add(5*time.Minute).Unix())
	_, err = s.dbConn.Exec(`UPDATE share_tokens SET scope_date_from = ?, scope_date_to = ? WHERE id = ?`, "2026-01-01", "2026-01-31", token.ID)
	if err != nil {
		t.Fatalf("set share scope: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/share/"+rawToken, strings.NewReader("password="+password))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "included note") {
		t.Fatalf("expected report to include in-range entry")
	}
	if strings.Contains(body, "excluded note") {
		t.Fatalf("expected report to exclude out-of-range entry")
	}
	if !strings.Contains(body, "Fatigue") {
		t.Fatalf("expected report to include trackable values")
	}
}

func assertShareSecurityHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("expected X-Robots-Tag=noindex, got %q", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected Referrer-Policy=no-referrer, got %q", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control=no-store, got %q", got)
	}
}

func seedShareToken(t *testing.T, s *Server, rawToken, password string, expiresAt int64) db.ShareToken {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	token, err := s.queries.CreateShareToken(context.Background(), db.CreateShareTokenParams{
		UserID:        1,
		TokenHash:     hashShareToken(rawToken),
		PasswordHash:  hashedPassword,
		ScopeDateFrom: sql.NullString{},
		ScopeDateTo:   sql.NullString{},
		ScopePrivate:  0,
		ExpiresAtUtc:  expiresAt,
		CreatedAtUtc:  time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("seed share token: %v", err)
	}

	return token
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
