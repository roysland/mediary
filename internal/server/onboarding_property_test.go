package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"roysland.me/symptomstracker/internal/db"

	"pgregory.net/rapid"
)

// Feature: app-feature-roadmap, Property 1: Onboarding redirect invariant
func TestProp_OnboardingRedirectInvariant(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rapid.Check(t, func(t *rapid.T) {
		mode := rapid.SampledFrom([]string{"absent", "complete", "other", "null"}).Draw(t, "mode")

		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = 'onboarding_complete'`, 1); err != nil {
			t.Fatalf("clear onboarding setting: %v", err)
		}

		switch mode {
		case "complete":
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, 'onboarding_complete', '1', 0)`, 1); err != nil {
				t.Fatalf("seed complete onboarding setting: %v", err)
			}
		case "other":
			value := rapid.String().Draw(t, "other_value")
			if value == "1" {
				value = "0"
			}
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, 'onboarding_complete', ?, 0)`, 1, value); err != nil {
				t.Fatalf("seed non-complete onboarding setting: %v", err)
			}
		case "null":
			if _, err := s.dbConn.Exec(`INSERT INTO settings (user_id, settings_key, settings_value, created_at_utc) VALUES (?, 'onboarding_complete', NULL, 0)`, 1); err != nil {
				t.Fatalf("seed null onboarding setting: %v", err)
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

		shouldRedirect := mode != "complete"
		if shouldRedirect {
			if rr.Code != http.StatusSeeOther {
				t.Fatalf("expected 303 redirect for mode %q, got %d", mode, rr.Code)
			}
			if location := rr.Header().Get("Location"); location != "/onboarding/1" {
				t.Fatalf("expected redirect location /onboarding/1 for mode %q, got %q", mode, location)
			}
			return
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for mode %q, got %d", mode, rr.Code)
		}
	})
}

func TestIsOnboardingComplete_NoSettingReturnsFalse(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	_, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = 'onboarding_complete'`, 1)
	if err != nil {
		t.Fatalf("clear onboarding setting: %v", err)
	}

	complete, err := s.isOnboardingComplete(t.Context(), 1)
	if err != nil {
		t.Fatalf("isOnboardingComplete returned error: %v", err)
	}
	if complete {
		t.Fatal("expected onboarding to be incomplete when setting is missing")
	}
}

func TestIsOnboardingComplete_QueryError(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DROP TABLE settings`); err != nil {
		t.Fatalf("drop settings table: %v", err)
	}

	complete, err := s.isOnboardingComplete(t.Context(), 1)
	if err == nil {
		t.Fatal("expected query error after dropping settings table")
	}
	if complete {
		t.Fatal("expected onboarding to be false on query error")
	}
	if err == sql.ErrNoRows {
		t.Fatal("expected a real query error, not sql.ErrNoRows")
	}
}

// Feature: app-feature-roadmap, Property 2: Onboarding completion persistence
func TestProp_OnboardingCompletionPersistence(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rapid.Check(t, func(t *rapid.T) {
		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = 'onboarding_complete'`, 1); err != nil {
			t.Fatalf("clear onboarding setting: %v", err)
		}

		step1Skip := rapid.Bool().Draw(t, "step1_skip")
		step2Skip := rapid.Bool().Draw(t, "step2_skip")
		step3Skip := rapid.Bool().Draw(t, "step3_skip")
		step4Skip := rapid.Bool().Draw(t, "step4_skip")
		step5Skip := rapid.Bool().Draw(t, "step5_skip")
		step2Language := rapid.SampledFrom([]string{"en", "no"}).Draw(t, "step2_language")

		runStep := func(step int, skip bool) {
			target := "/onboarding/" + strconv.Itoa(step)
			var body *strings.Reader

			if skip {
				target += "/skip"
				body = strings.NewReader("")
			} else if step == 2 {
				body = strings.NewReader("language=" + step2Language)
			} else {
				body = strings.NewReader("")
			}

			req := httptest.NewRequest(http.MethodPost, target, body)
			cookieResp := httptest.NewRecorder()
			if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
				t.Fatalf("set authenticated user cookie: %v", err)
			}
			for _, cookie := range cookieResp.Result().Cookies() {
				req.AddCookie(cookie)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if rr.Code != http.StatusSeeOther {
				t.Fatalf("expected 303 for step %d (skip=%t), got %d", step, skip, rr.Code)
			}

			if step < 5 {
				expected := "/onboarding/" + strconv.Itoa(step+1)
				if location := rr.Header().Get("Location"); location != expected {
					t.Fatalf("expected redirect %q for step %d (skip=%t), got %q", expected, step, skip, location)
				}
				return
			}

			if location := rr.Header().Get("Location"); location != "/" {
				t.Fatalf("expected redirect to / for final step, got %q", location)
			}
		}

		runStep(1, step1Skip)
		runStep(2, step2Skip)
		runStep(3, step3Skip)
		runStep(4, step4Skip)
		runStep(5, step5Skip)

		setting, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: "onboarding_complete"})
		if err != nil {
			t.Fatalf("fetch onboarding_complete setting: %v", err)
		}

		if !setting.SettingsValue.Valid || setting.SettingsValue.String != "1" {
			t.Fatalf("expected onboarding_complete=1, got %#v", setting.SettingsValue)
		}
	})
}

// Feature: app-feature-roadmap, Property 3: Onboarding skip advances step
func TestProp_OnboardingSkipAdvancesStep(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	rapid.Check(t, func(t *rapid.T) {
		if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = 'onboarding_complete'`, 1); err != nil {
			t.Fatalf("clear onboarding setting: %v", err)
		}

		step := rapid.IntRange(1, 5).Draw(t, "step")
		req := httptest.NewRequest(http.MethodPost, "/onboarding/"+strconv.Itoa(step)+"/skip", strings.NewReader(""))
		cookieResp := httptest.NewRecorder()
		if err := s.authSessions.SetAuthenticatedUser(cookieResp, 1); err != nil {
			t.Fatalf("set authenticated user cookie: %v", err)
		}
		for _, cookie := range cookieResp.Result().Cookies() {
			req.AddCookie(cookie)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Fatalf("expected 303 for step %d skip, got %d", step, rr.Code)
		}

		expected := "/"
		if step < 5 {
			expected = "/onboarding/" + strconv.Itoa(step+1)
		}

		if location := rr.Header().Get("Location"); location != expected {
			t.Fatalf("expected redirect %q for step %d skip, got %q", expected, step, location)
		}
	})
}
