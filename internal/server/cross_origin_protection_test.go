package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCrossOriginProtectionRejectsUntrustedCrossSiteWrite(t *testing.T) {
	h := withCrossOriginProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), Config{})

	req := httptest.NewRequest(http.MethodPost, "http://internal.local/settings", strings.NewReader("x=1"))
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestCrossOriginProtectionAllowsTrustedOriginWrite(t *testing.T) {
	h := withCrossOriginProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), Config{CSRFTrustedOrigins: []string{"https://app.example.com"}})

	req := httptest.NewRequest(http.MethodPost, "http://internal.local/settings", strings.NewReader("x=1"))
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestCrossOriginProtectionAllowsSafeMethods(t *testing.T) {
	h := withCrossOriginProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), Config{})

	req := httptest.NewRequest(http.MethodGet, "http://internal.local/settings", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
