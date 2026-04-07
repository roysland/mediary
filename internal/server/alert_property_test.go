package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
	"roysland.me/symptomstracker/internal/db"
)

// Feature: app-feature-roadmap, Property 4: Alert banner visibility
func TestProp_AlertBannerVisibility(t *testing.T) {
	if activeAlertVersion == "" {
		t.Skip("no active alert configured")
	}

	s := newHomeEntriesHTTPTestServer(t)
	key := "alert_dismissed_" + activeAlertVersion

	rapid.Check(t, func(t *rapid.T) {
		mode := rapid.SampledFrom([]string{"absent", "dismissed", "other", "null"}).Draw(t, "mode")

		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = ?`, 1, key); err != nil {
			t.Fatalf("clear alert setting: %v", err)
		}

		switch mode {
		case "dismissed":
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, ?, '1', 0)`, 1, key); err != nil {
				t.Fatalf("seed dismissed alert setting: %v", err)
			}
		case "other":
			value := rapid.StringMatching(`[a-z0-9_-]{1,12}`).Draw(t, "other_value")
			if value == "1" {
				value = "0"
			}
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, ?, ?, 0)`, 1, key, value); err != nil {
				t.Fatalf("seed other alert setting: %v", err)
			}
		case "null":
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, ?, NULL, 0)`, 1, key); err != nil {
				t.Fatalf("seed null alert setting: %v", err)
			}
		case "absent":
			// Keep row absent.
		default:
			t.Fatalf("unknown mode: %q", mode)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		cookieResp := httptest.NewRecorder()
		if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
			t.Fatalf("set authenticated user cookie: %v", err)
		}
		for _, cookie := range cookieResp.Result().Cookies() {
			req.AddCookie(cookie)
		}

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		hasBanner := strings.Contains(rr.Body.String(), `id="alert-banner"`)
		expectedBanner := mode != "dismissed"
		if hasBanner != expectedBanner {
			t.Fatalf("expected banner=%t for mode %q, got %t", expectedBanner, mode, hasBanner)
		}
	})
}

// Feature: app-feature-roadmap, Property 5: Alert dismissal persistence
func TestProp_AlertDismissalPersistence(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rapid.Check(t, func(t *rapid.T) {
		version := rapid.StringMatching(`[a-z0-9][a-z0-9_-]{3,20}`).Draw(t, "version")
		key := "alert_dismissed_" + version

		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = ?`, 1, key); err != nil {
			t.Fatalf("clear versioned alert setting: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/alert/"+version+"/dismiss", strings.NewReader(""))
		req.Header.Set("HX-Request", "true")
		cookieResp := httptest.NewRecorder()
		if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
			t.Fatalf("set authenticated user cookie: %v", err)
		}
		for _, cookie := range cookieResp.Result().Cookies() {
			req.AddCookie(cookie)
		}

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if rr.Body.Len() != 0 {
			t.Fatalf("expected empty body, got %q", rr.Body.String())
		}

		setting, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: key})
		if err != nil {
			t.Fatalf("fetch alert dismissal setting: %v", err)
		}
		if !setting.SettingsValue.Valid || setting.SettingsValue.String != "1" {
			t.Fatalf("expected setting %q to be 1, got %#v", key, setting.SettingsValue)
		}
	})
}

// Feature: app-feature-roadmap, Property 6: Alert version isolation
func TestProp_AlertVersionIsolation(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rapid.Check(t, func(t *rapid.T) {
		versionA := rapid.StringMatching(`[a-z0-9][a-z0-9_-]{3,20}`).Draw(t, "version_a")
		versionB := rapid.StringMatching(`[a-z0-9][a-z0-9_-]{3,20}`).Draw(t, "version_b")
		if versionA == versionB {
			return
		}

		keyA := "alert_dismissed_" + versionA
		keyB := "alert_dismissed_" + versionB

		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key IN (?, ?)`, 1, keyA, keyB); err != nil {
			t.Fatalf("clear alert settings: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/alert/"+versionA+"/dismiss", strings.NewReader(""))
		req.Header.Set("HX-Request", "true")
		cookieResp := httptest.NewRecorder()
		if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
			t.Fatalf("set authenticated user cookie: %v", err)
		}
		for _, cookie := range cookieResp.Result().Cookies() {
			req.AddCookie(cookie)
		}

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		settingA, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: keyA})
		if err != nil {
			t.Fatalf("fetch alert setting for version A: %v", err)
		}
		if !settingA.SettingsValue.Valid || settingA.SettingsValue.String != "1" {
			t.Fatalf("expected version A setting to be 1, got %#v", settingA.SettingsValue)
		}

		_, err = s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: keyB})
		if err != sql.ErrNoRows {
			t.Fatalf("expected no setting for version B, got err=%v", err)
		}
	})
}
