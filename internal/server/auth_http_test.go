package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"roysland.me/symptomstracker/internal/auth"
)

// TestWithSessionRequired_UnauthBrowserRequestRedirects verifies that an
// unauthenticated plain browser GET to a protected route is redirected to /auth.
func TestWithSessionRequired_UnauthBrowserRequestRedirects(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/auth" {
		t.Fatalf("expected redirect to /auth, got %q", loc)
	}
}

// TestWithSessionRequired_UnauthHTMXRequestReturnsUnauthorized verifies that an
// unauthenticated HTMX request to a protected route receives a 401 instead of a redirect.
func TestWithSessionRequired_UnauthHTMXRequestReturnsUnauthorized(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// TestWithSessionRequired_UnauthJSONRequestReturnsUnauthorized verifies that an
// unauthenticated request that accepts JSON to a protected route receives a 401 instead of a redirect.
func TestWithSessionRequired_UnauthJSONRequestReturnsUnauthorized(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// TestWithSessionRequired_PublicRoutesAccessibleWithoutAuth verifies that the
// /auth page is accessible without a session.
func TestWithSessionRequired_PublicRoutesAccessibleWithoutAuth(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /auth without session, got %d", rr.Code)
	}
}

// TestAuthPage_RedirectsAuthenticatedUser verifies that a user who already has a
// valid session is sent back to / when they visit /auth.
func TestAuthPage_RedirectsAuthenticatedUser(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodGet, "/auth", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for authenticated user on /auth, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

// TestLogout_BrowserClearsSessionAndRedirects verifies that posting to /auth/logout
// clears the session cookie and redirects the browser to /auth.
func TestLogout_BrowserClearsSessionAndRedirects(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after logout, got %d: %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/auth" {
		t.Fatalf("expected redirect to /auth after logout, got %q", loc)
	}

	// The session cookie should be cleared (MaxAge < 0 or empty value).
	var cleared bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			if c.MaxAge < 0 || c.Value == "" {
				cleared = true
			}
			break
		}
	}
	if !cleared {
		t.Fatalf("expected session cookie to be cleared after logout")
	}
}

// TestLogout_AJAXReturnsJSON verifies that an HTMX logout request returns JSON
// instead of an HTML redirect.
func TestLogout_AJAXReturnsJSON(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodPost, "/auth/logout", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for AJAX logout, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected JSON content-type for AJAX logout, got %q", ct)
	}
}

// TestLogout_UnauthenticatedSessionStillRedirects verifies that posting to
// /auth/logout without a valid session still redirects to /auth (it is a public route).
func TestLogout_UnauthenticatedSessionStillRedirects(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after logout with no session, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/auth" {
		t.Fatalf("expected redirect to /auth, got %q", loc)
	}
}

// TestFinishRegistration_MissingCeremonyCookieReturnsBadRequest verifies that
// POSTing to /webauthn/register/verify without a ceremony cookie returns 400.
func TestFinishRegistration_MissingCeremonyCookieReturnsBadRequest(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/webauthn/register/verify", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when ceremony cookie is absent, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestFinishLogin_MissingCeremonyCookieReturnsBadRequest verifies that
// POSTing to /webauthn/login/verify without a ceremony cookie returns 400.
func TestFinishLogin_MissingCeremonyCookieReturnsBadRequest(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/webauthn/login/verify", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when ceremony cookie is absent, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestFinishAddPasskey_MissingCeremonyCookieReturnsBadRequest verifies that
// POSTing to /webauthn/passkeys/verify without a ceremony cookie returns 400.
func TestFinishAddPasskey_MissingCeremonyCookieReturnsBadRequest(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodPost, "/webauthn/passkeys/verify", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when ceremony cookie is absent, got %d: %s", rr.Code, rr.Body.String())
	}
}
