//go:build !e2e

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestE2ELogin_NotRegisteredWithoutE2EBuildTag(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)
	s.cfg.AppEnv = "test"
	s.cfg.E2EAuthToken = "configured-token"

	req := httptest.NewRequest(http.MethodGet, "/auth/e2e/login?token=configured-token", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when e2e login route is not compiled in, got %d", rr.Code)
	}
}
