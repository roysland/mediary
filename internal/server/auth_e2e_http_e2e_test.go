//go:build e2e

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"roysland.me/symptomstracker/internal/auth"
)

func TestE2ELogin_UnavailableInProductionMode(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "production"
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=playwright-token", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 in production mode, got %d", rr.Code)
	}
}

func TestE2ELogin_SetsSessionAndRedirectsOnlyInTestMode(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "test"
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=playwright-token&redirect=%2Fentries", nil)
	req.RemoteAddr = "127.0.0.1:34567"
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

func TestE2ELogin_DisabledWhenTokenMissing(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "test"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=test", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when e2e login token is missing, got %d", rr.Code)
	}
}

func TestE2ELogin_RejectsNonLocalhostRequests(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "test"
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=playwright-token", nil)
	req.RemoteAddr = "198.51.100.10:34567"
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-localhost request, got %d", rr.Code)
	}
}

func TestE2ELogin_RejectsWrongToken(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "test"
	s.cfg.E2EAuthToken = "playwright-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=wrong", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong token, got %d", rr.Code)
	}
}
