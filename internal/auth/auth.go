package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SessionCookieName  = "st_session"
	CeremonyCookieName = "st_webauthn"
	defaultSessionTTL  = 30 * 24 * time.Hour
)

type User struct {
	ID int64
}

type SessionManager struct {
	secret      []byte
	secure      bool
	sessionTTL  time.Duration
	cookiePath  string
	cookieSame  http.SameSite
	cookieHost  string
	ceremonyTTL time.Duration
}

type sessionPayload struct {
	UserID  int64 `json:"uid"`
	Expires int64 `json:"exp"`
}

var (
	defaultSessionManagerMu sync.RWMutex
	defaultSessionManager   *SessionManager
)

func NewSessionManager(secret string, secure bool) (*SessionManager, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		return nil, errors.New("auth session secret must be at least 32 characters")
	}

	return &SessionManager{
		secret:      []byte(secret),
		secure:      secure,
		sessionTTL:  defaultSessionTTL,
		cookiePath:  "/",
		cookieSame:  http.SameSiteLaxMode,
		ceremonyTTL: 10 * time.Minute,
	}, nil
}

func SetDefaultSessionManager(mgr *SessionManager) {
	defaultSessionManagerMu.Lock()
	defer defaultSessionManagerMu.Unlock()
	defaultSessionManager = mgr
}

func CurrentUser(r *http.Request) *User {
	defaultSessionManagerMu.RLock()
	mgr := defaultSessionManager
	defaultSessionManagerMu.RUnlock()
	if mgr == nil {
		return nil
	}

	uid, ok := mgr.UserIDFromRequest(r)
	if !ok {
		return nil
	}

	return &User{ID: uid}
}

func (m *SessionManager) UserIDFromRequest(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return 0, false
	}

	payload, err := m.verifySignedPayload(cookie.Value)
	if err != nil {
		return 0, false
	}

	var session sessionPayload
	if err := json.Unmarshal(payload, &session); err != nil {
		return 0, false
	}

	if session.UserID <= 0 {
		return 0, false
	}

	if session.Expires <= time.Now().Unix() {
		return 0, false
	}

	return session.UserID, true
}

func (m *SessionManager) SetAuthenticatedUser(w http.ResponseWriter, userID int64) error {
	if userID <= 0 {
		return errors.New("user id must be positive")
	}

	payload, err := json.Marshal(sessionPayload{
		UserID:  userID,
		Expires: time.Now().Add(m.sessionTTL).Unix(),
	})
	if err != nil {
		return fmt.Errorf("marshal session payload: %w", err)
	}

	signed := m.signPayload(payload)
	m.setCookie(w, SessionCookieName, signed, m.sessionTTL)
	return nil
}

func (m *SessionManager) ClearSession(w http.ResponseWriter) {
	m.setCookie(w, SessionCookieName, "", -time.Hour)
}

func (m *SessionManager) NewCeremonyID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate ceremony id: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (m *SessionManager) SetCeremonyID(w http.ResponseWriter, id string) {
	m.setCookie(w, CeremonyCookieName, id, m.ceremonyTTL)
}

func (m *SessionManager) CeremonyIDFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(CeremonyCookieName)
	if err != nil {
		return "", false
	}

	id := strings.TrimSpace(cookie.Value)
	if id == "" {
		return "", false
	}

	return id, true
}

func (m *SessionManager) ClearCeremonyID(w http.ResponseWriter) {
	m.setCookie(w, CeremonyCookieName, "", -time.Hour)
}

func (m *SessionManager) signPayload(payload []byte) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write(payload)
	sig := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (m *SessionManager) verifySignedPayload(value string) ([]byte, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid signed payload format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, errors.New("signature mismatch")
	}

	return payload, nil
}

func (m *SessionManager) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	maxAge := int(ttl / time.Second)
	if ttl < 0 {
		maxAge = -1
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     m.cookiePath,
		Domain:   m.cookieHost,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.cookieSame,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(ttl),
	})
}

func ParseOptionalUserID(raw string) (int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}

	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}

	return parsed, true
}
