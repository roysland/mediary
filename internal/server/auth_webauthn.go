package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
	"roysland.me/symptomstracker/internal/auth"
	"roysland.me/symptomstracker/internal/db"
)

const (
	ceremonyKindRegister   = "register"
	ceremonyKindLogin      = "login"
	ceremonyKindAddPasskey = "add-passkey"
)

type webauthnCeremony struct {
	Kind      string
	UserID    int64
	Session   webauthnlib.SessionData
	CreatedAt int64
}

type webauthnUser struct {
	user        db.User
	credentials []db.WebauthnCredential
}

func (u webauthnUser) WebAuthnID() []byte {
	return u.user.WebauthnUserID
}

func (u webauthnUser) WebAuthnName() string {
	if u.user.DisplayName.Valid && strings.TrimSpace(u.user.DisplayName.String) != "" {
		return strings.TrimSpace(u.user.DisplayName.String)
	}

	return fmt.Sprintf("user-%d", u.user.ID)
}

func (u webauthnUser) WebAuthnDisplayName() string {
	if u.user.DisplayName.Valid && strings.TrimSpace(u.user.DisplayName.String) != "" {
		return strings.TrimSpace(u.user.DisplayName.String)
	}

	return "Anonymous"
}

func (u webauthnUser) WebAuthnCredentials() []webauthnlib.Credential {
	result := make([]webauthnlib.Credential, 0, len(u.credentials))
	for _, cred := range u.credentials {
		result = append(result, webauthnlib.Credential{
			ID:        cred.CredentialID,
			PublicKey: cred.PublicKey,
			Transport: parseTransportList(cred.Transports),
			Authenticator: webauthnlib.Authenticator{
				SignCount: uint32(cred.SignCount),
			},
			Flags: parseCredentialFlags(cred.Flags),
		})
	}

	return result
}

func (u webauthnUser) credentialDescriptors() []protocol.CredentialDescriptor {
	credentials := u.WebAuthnCredentials()
	descriptors := make([]protocol.CredentialDescriptor, 0, len(credentials))
	for _, credential := range credentials {
		descriptors = append(descriptors, credential.Descriptor())
	}
	return descriptors
}

func (s *Server) authPage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	if current := auth.CurrentUser(r); current != nil && current.ID > 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.renderPage(w, r, "auth_title", "auth_content", map[string]any{})
}

func (s *Server) beginRegistration(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	if _, ok := requireAnonymous(w, r); !ok {
		return
	}

	var body struct {
		DisplayName string `json:"display_name"`
		DeviceName  string `json:"device_name"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	displayName := strings.TrimSpace(body.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(body.DeviceName)
	}
	if displayName == "" {
		displayName = "Anonymous"
	}

	userIDBytes, err := s.authSessions.NewCeremonyID()
	if err != nil {
		respondInternalError(w, r, "Failed to initialize passkey registration")
		return
	}

	createdUser, err := s.queries.CreateUser(r.Context(), db.CreateUserParams{
		CreatedAtUtc:   time.Now().Unix(),
		WebauthnUserID: []byte(userIDBytes),
		DisplayName:    sql.NullString{String: displayName, Valid: displayName != ""},
	})
	if err != nil {
		respondInternalError(w, r, "Failed to create account")
		return
	}

	user := webauthnUser{user: createdUser}

	options, sessionData, err := s.webauthn.BeginRegistration(
		user,
		webauthnlib.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthnlib.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			RequireResidentKey:      protocol.ResidentKeyRequired(),
			UserVerification:        protocol.VerificationPreferred,
		}),
		webauthnlib.WithPublicKeyCredentialHints([]protocol.PublicKeyCredentialHints{
			protocol.PublicKeyCredentialHintClientDevice,
			protocol.PublicKeyCredentialHintHybrid,
		}),
	)
	if err != nil {
		respondInternalError(w, r, "Failed to begin passkey registration")
		return
	}

	if err := s.startCeremony(w, ceremonyKindRegister, createdUser.ID, sessionData); err != nil {
		respondInternalError(w, r, "Failed to start registration ceremony")
		return
	}

	respondJSON(w, http.StatusOK, options)
}

func (s *Server) finishRegistration(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	state, ok := s.consumeCeremony(w, r, ceremonyKindRegister)
	if !ok {
		respondBadRequest(w, r, "Registration session is missing or expired")
		return
	}

	user, err := s.loadWebauthnUser(r.Context(), state.UserID)
	if err != nil {
		respondBadRequest(w, r, "Account not found")
		return
	}

	credential, err := s.webauthn.FinishRegistration(user, state.Session, r)
	if err != nil {
		respondBadRequest(w, r, "Invalid passkey registration response")
		return
	}

	if err := s.storeCredential(r.Context(), user.user.ID, credential); err != nil {
		respondInternalError(w, r, "Failed to save passkey")
		return
	}

	if err := s.authSessions.SetAuthenticatedUser(w, user.user.ID); err != nil {
		respondInternalError(w, r, "Failed to create session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"redirect": "/",
	})
}

func (s *Server) beginLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	conditional := r.URL.Query().Get("conditional") == "1"

	var (
		assertion   *protocol.CredentialAssertion
		sessionData *webauthnlib.SessionData
		err         error
	)

	loginOptions := []webauthnlib.LoginOption{
		webauthnlib.WithUserVerification(protocol.VerificationPreferred),
		webauthnlib.WithAssertionPublicKeyCredentialHints([]protocol.PublicKeyCredentialHints{
			protocol.PublicKeyCredentialHintClientDevice,
			protocol.PublicKeyCredentialHintHybrid,
		}),
	}

	if conditional {
		assertion, sessionData, err = s.webauthn.BeginDiscoverableMediatedLogin(protocol.MediationConditional, loginOptions...)
	} else {
		assertion, sessionData, err = s.webauthn.BeginDiscoverableLogin(loginOptions...)
	}
	if err != nil {
		respondInternalError(w, r, "Failed to begin passkey login")
		return
	}

	if err := s.startCeremony(w, ceremonyKindLogin, 0, sessionData); err != nil {
		respondInternalError(w, r, "Failed to start login ceremony")
		return
	}

	respondJSON(w, http.StatusOK, assertion)
}

func (s *Server) finishLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	state, ok := s.consumeCeremony(w, r, ceremonyKindLogin)
	if !ok {
		respondBadRequest(w, r, "Login session is missing or expired")
		return
	}

	user, credential, err := s.webauthn.FinishPasskeyLogin(
		s.resolveDiscoverableUser,
		state.Session,
		r,
	)
	if err != nil {
		log.Printf("webauthn login failed: %v", err)
		respondBadRequest(w, r, "Invalid login response")
		return
	}

	resolved, ok := user.(webauthnUser)
	if !ok {
		respondInternalError(w, r, "Unexpected passkey user type")
		return
	}

	if err := s.updateCredentialSignCount(r.Context(), credential); err != nil {
		respondInternalError(w, r, "Failed to update passkey state")
		return
	}

	if err := s.authSessions.SetAuthenticatedUser(w, resolved.user.ID); err != nil {
		respondInternalError(w, r, "Failed to create session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"redirect": "/",
	})
}

func (s *Server) beginAddPasskey(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	user, err := s.loadWebauthnUser(r.Context(), userID)
	if err != nil {
		respondInternalError(w, r, "Failed to load account")
		return
	}

	options, sessionData, err := s.webauthn.BeginRegistration(
		user,
		webauthnlib.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthnlib.WithExclusions(user.credentialDescriptors()),
		webauthnlib.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			RequireResidentKey:      protocol.ResidentKeyRequired(),
			UserVerification:        protocol.VerificationPreferred,
		}),
		webauthnlib.WithPublicKeyCredentialHints([]protocol.PublicKeyCredentialHints{
			protocol.PublicKeyCredentialHintClientDevice,
			protocol.PublicKeyCredentialHintHybrid,
		}),
	)
	if err != nil {
		respondInternalError(w, r, "Failed to begin passkey registration")
		return
	}

	if err := s.startCeremony(w, ceremonyKindAddPasskey, userID, sessionData); err != nil {
		respondInternalError(w, r, "Failed to start passkey ceremony")
		return
	}

	respondJSON(w, http.StatusOK, options)
}

func (s *Server) finishAddPasskey(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	state, ok := s.consumeCeremony(w, r, ceremonyKindAddPasskey)
	if !ok {
		respondBadRequest(w, r, "Passkey ceremony is missing or expired")
		return
	}

	requestUserID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if requestUserID != state.UserID {
		respondForbidden(w, r, "Cannot add passkey for another account")
		return
	}

	user, err := s.loadWebauthnUser(r.Context(), state.UserID)
	if err != nil {
		respondInternalError(w, r, "Failed to load account")
		return
	}

	credential, err := s.webauthn.FinishRegistration(user, state.Session, r)
	if err != nil {
		respondBadRequest(w, r, "Invalid passkey registration response")
		return
	}

	if err := s.storeCredential(r.Context(), state.UserID, credential); err != nil {
		respondInternalError(w, r, "Failed to save passkey")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	s.authSessions.ClearSession(w)
	if classifyRequest(r).IsAJAX {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	http.Redirect(w, r, "/auth", http.StatusSeeOther)
}

func requireAnonymous(w http.ResponseWriter, r *http.Request) (int64, bool) {
	current := auth.CurrentUser(r)
	if current != nil && current.ID > 0 {
		respondBadRequest(w, r, "You are already authenticated")
		return current.ID, false
	}
	return 0, true
}

func (s *Server) startCeremony(w http.ResponseWriter, kind string, userID int64, session *webauthnlib.SessionData) error {
	if session == nil {
		return errors.New("session data is nil")
	}

	id, err := s.authSessions.NewCeremonyID()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	s.ceremonyMu.Lock()
	for key, ceremony := range s.ceremonies {
		if now-ceremony.CreatedAt > 15*60 {
			delete(s.ceremonies, key)
		}
	}
	s.ceremonies[id] = webauthnCeremony{
		Kind:      kind,
		UserID:    userID,
		Session:   *session,
		CreatedAt: now,
	}
	s.ceremonyMu.Unlock()

	s.authSessions.SetCeremonyID(w, id)
	return nil
}

func (s *Server) consumeCeremony(w http.ResponseWriter, r *http.Request, expectedKind string) (webauthnCeremony, bool) {
	id, ok := s.authSessions.CeremonyIDFromRequest(r)
	if !ok {
		return webauthnCeremony{}, false
	}

	s.ceremonyMu.Lock()
	state, found := s.ceremonies[id]
	if found {
		delete(s.ceremonies, id)
	}
	s.ceremonyMu.Unlock()

	s.authSessions.ClearCeremonyID(w)
	if !found || state.Kind != expectedKind {
		return webauthnCeremony{}, false
	}

	return state, true
}

func (s *Server) resolveDiscoverableUser(rawID, userHandle []byte) (webauthnlib.User, error) {
	user, err := s.queries.GetUserByWebauthnUserID(context.Background(), userHandle)
	if err != nil {
		return nil, fmt.Errorf("user not found by handle: %w", err)
	}

	loaded, err := s.loadWebauthnUser(context.Background(), user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user credentials: %w", err)
	}

	for _, credential := range loaded.credentials {
		if bytes.Equal(credential.CredentialID, rawID) {
			return loaded, nil
		}
	}

	return nil, fmt.Errorf("credential ID %x not found in user's credentials", rawID)
}

func (s *Server) loadWebauthnUser(ctx context.Context, userID int64) (webauthnUser, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return webauthnUser{}, err
	}

	credentials, err := s.queries.ListWebauthnCredentialsByUser(ctx, user.ID)
	if err != nil {
		return webauthnUser{}, err
	}

	return webauthnUser{user: user, credentials: credentials}, nil
}

func (s *Server) storeCredential(ctx context.Context, userID int64, credential *webauthnlib.Credential) error {
	if credential == nil {
		return errors.New("credential is nil")
	}

	flagsJSON, err := json.Marshal(credential.Flags)
	if err != nil {
		return fmt.Errorf("marshal flags: %w", err)
	}

	_, err = s.queries.CreateWebauthnCredential(ctx, db.CreateWebauthnCredentialParams{
		UserID:       userID,
		CredentialID: credential.ID,
		PublicKey:    credential.PublicKey,
		SignCount:    int64(credential.Authenticator.SignCount),
		Transports:   encodeTransportList(credential.Transport),
		Flags:        sql.NullString{String: string(flagsJSON), Valid: true},
		CreatedAtUtc: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) updateCredentialSignCount(ctx context.Context, credential *webauthnlib.Credential) error {
	if credential == nil {
		return errors.New("credential is nil")
	}

	return s.queries.UpdateWebauthnCredentialSignCount(ctx, db.UpdateWebauthnCredentialSignCountParams{
		CredentialID: credential.ID,
		SignCount:    int64(credential.Authenticator.SignCount),
	})
}

func parseTransportList(raw sql.NullString) []protocol.AuthenticatorTransport {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw.String), &values); err != nil {
		return nil
	}

	transports := make([]protocol.AuthenticatorTransport, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		transports = append(transports, protocol.AuthenticatorTransport(value))
	}

	return transports
}

func encodeTransportList(transports []protocol.AuthenticatorTransport) sql.NullString {
	if len(transports) == 0 {
		return sql.NullString{}
	}

	values := make([]string, 0, len(transports))
	for _, transport := range transports {
		values = append(values, string(transport))
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return sql.NullString{}
	}

	return sql.NullString{String: string(encoded), Valid: true}
}

func parseCredentialFlags(raw sql.NullString) webauthnlib.CredentialFlags {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return webauthnlib.CredentialFlags{}
	}

	var flags webauthnlib.CredentialFlags
	if err := json.Unmarshal([]byte(raw.String), &flags); err != nil {
		return webauthnlib.CredentialFlags{}
	}

	return flags
}
