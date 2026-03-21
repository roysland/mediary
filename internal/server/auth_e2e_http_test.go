package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"roysland.me/symptomstracker/internal/auth"
)

func TestE2ELogin_DisabledWhenTokenMissing(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=test", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when e2e login is disabled, got %d", rr.Code)
	}
}

func TestE2ELogin_SetsSessionAndRedirects(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.devMode = true
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=playwright-token&redirect=%2Fentries", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect from e2e login, got %d", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/entries" {
		t.Fatalf("expected redirect to /entries, got %q", location)
	}

	var hasSessionCookie bool
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName {
			hasSessionCookie = true
			break
		}
	}
	if !hasSessionCookie {
		t.Fatalf("expected auth session cookie %q to be set", auth.SessionCookieName)
	}
}

func TestE2ELogin_RejectsWrongToken(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.devMode = true
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=wrong", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong token, got %d", rr.Code)
	}
}
