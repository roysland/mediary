//go:build e2e

package server

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

const e2eWebAuthnUserID = "playwright-e2e-user"

const e2eLoginPath = "/auth/e2e/login"

func (s *Server) registerE2ERoutes() {
	s.mux.HandleFunc(e2eLoginPath, s.e2eLogin)
}

func isPublicE2ERoute(path string) bool {
	return path == e2eLoginPath
}

// e2eLogin is a test-only authentication bypass for browser E2E runs.
// It must only be enabled in APP_ENV=test and must never be exposed in production.
func (s *Server) e2eLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondNotFound(w, r, "Not found")
		return
	}

	if !s.isE2ERequestAuthorized(r) {
		respondNotFound(w, r, "Not found")
		return
	}

	user, err := s.ensureE2EUser(r.Context())
	if err != nil {
		respondInternalError(w, r, "Failed to initialize e2e user")
		return
	}

	if err := s.authSessions.SetAuthenticatedUser(w, user.ID); err != nil {
		respondInternalError(w, r, "Failed to create session")
		return
	}

	redirectTo := strings.TrimSpace(r.URL.Query().Get("redirect"))
	if !strings.HasPrefix(redirectTo, "/") || strings.HasPrefix(redirectTo, "//") {
		redirectTo = "/"
	}

	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (s *Server) isE2ERequestAuthorized(r *http.Request) bool {
	if s.cfg.AppEnv != "test" {
		return false
	}

	token := strings.TrimSpace(s.cfg.E2EAuthToken)
	if token == "" {
		return false
	}

	if !isLoopbackRemoteAddr(r.RemoteAddr) {
		return false
	}

	requestToken := strings.TrimSpace(r.URL.Query().Get("token"))
	return subtle.ConstantTimeCompare([]byte(requestToken), []byte(token)) == 1
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host := strings.TrimSpace(remoteAddr)
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsedHost
	}

	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	return addr.IsLoopback()
}

func (s *Server) ensureE2EUser(ctx context.Context) (db.User, error) {
	user, err := s.queries.GetUserByWebauthnUserID(ctx, []byte(e2eWebAuthnUserID))
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.User{}, err
	}

	now := time.Now().UTC().Unix()
	created, createErr := s.queries.CreateUser(ctx, db.CreateUserParams{
		CreatedAtUtc:   now,
		WebauthnUserID: []byte(e2eWebAuthnUserID),
		DisplayName: sql.NullString{
			String: "Playwright E2E",
			Valid:  true,
		},
	})
	if createErr == nil {
		return created, nil
	}

	// In case of concurrent requests, another request may have created it first.
	user, retryErr := s.queries.GetUserByWebauthnUserID(ctx, []byte(e2eWebAuthnUserID))
	if retryErr == nil {
		return user, nil
	}

	return db.User{}, createErr
}
