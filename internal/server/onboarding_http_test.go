package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"roysland.me/symptomstracker/internal/db"
)

func TestOnboardingPreview_IsPublic(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/onboarding/preview", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "How to get started") {
		t.Fatalf("expected onboarding preview content, got: %s", rr.Body.String())
	}
}

func TestOnboardingStep_RequiresSession(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/onboarding/1", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/auth" {
		t.Fatalf("expected redirect to /auth, got %q", loc)
	}
}

func TestOnboardingStep_IncompleteUserSeesStepPage(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = 1 AND settings_key = 'onboarding_complete'`); err != nil {
		t.Fatalf("delete onboarding setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodGet, "/onboarding/1", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `action="/onboarding/1"`) {
		t.Fatalf("expected onboarding step form in response")
	}
}

func TestOnboardingStep_CompleteUserRedirectsHome(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	req := authedRequest(t, s, http.MethodGet, "/onboarding/1", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

func TestOnboardingStep2Post_PersistsLanguageAndAdvances(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = 1 AND settings_key = 'onboarding_complete'`); err != nil {
		t.Fatalf("delete onboarding setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodPost, "/onboarding/2", strings.NewReader("language=no"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/onboarding/3" {
		t.Fatalf("expected redirect to /onboarding/3, got %q", loc)
	}

	setting, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: "language"})
	if err != nil {
		t.Fatalf("fetch language setting: %v", err)
	}
	if !setting.SettingsValue.Valid || setting.SettingsValue.String != "no" {
		t.Fatalf("expected language=no, got %#v", setting.SettingsValue)
	}
}

func TestOnboardingSkip_AdvancesToNextStep(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = 1 AND settings_key = 'onboarding_complete'`); err != nil {
		t.Fatalf("delete onboarding setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodPost, "/onboarding/3/skip", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/onboarding/4" {
		t.Fatalf("expected redirect to /onboarding/4, got %q", loc)
	}
}

func TestOnboardingFinalStepSkip_MarksCompleteAndRedirectsHome(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = 1 AND settings_key = 'onboarding_complete'`); err != nil {
		t.Fatalf("delete onboarding setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodPost, "/onboarding/5/skip", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}

	setting, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: "onboarding_complete"})
	if err != nil {
		t.Fatalf("fetch onboarding_complete setting: %v", err)
	}
	if !setting.SettingsValue.Valid || setting.SettingsValue.String != "1" {
		t.Fatalf("expected onboarding_complete=1, got %#v", setting.SettingsValue)
	}
}

func TestOnboardingStep2Post_RejectsInvalidLanguage(t *testing.T) {
	s := newHomeEntriesHTTPTestServer(t)

	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = 1 AND settings_key = 'onboarding_complete'`); err != nil {
		t.Fatalf("delete onboarding setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodPost, "/onboarding/2", strings.NewReader("language=sv"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	_, err := s.queries.GetSetting(t.Context(), db.GetSettingParams{UserID: 1, SettingsKey: "language"})
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("expected no language write, got unexpected error: %v", err)
	}
}
