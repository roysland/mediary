package server

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

const e2eWebAuthnUserID = "playwright-e2e-user"

func (s *Server) e2eLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	token := strings.TrimSpace(s.cfg.E2EAuthToken)
	if !s.devMode || token == "" {
		respondNotFound(w, r, "Not found")
		return
	}

	requestToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if subtle.ConstantTimeCompare([]byte(requestToken), []byte(token)) != 1 {
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
