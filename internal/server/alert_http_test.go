package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeAlertBannerRendersHTMXDismissContract(t *testing.T) {
	if activeAlertVersion == "" {
		t.Skip("no active alert configured")
	}

	s := newHomeEntriesHTTPTestServer(t)
	key := "alert_dismissed_" + activeAlertVersion
	if _, err := s.dbConn.Exec(`DELETE FROM settings WHERE user_id = ? AND settings_key = ?`, 1, key); err != nil {
		t.Fatalf("clear alert setting: %v", err)
	}

	req := authedRequest(t, s, http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `id="alert-banner"`) {
		t.Fatalf("expected alert banner on home page")
	}
	if !strings.Contains(body, `hx-post="/alert/`+activeAlertVersion+`/dismiss"`) {
		t.Fatalf("expected dismiss hx-post route in banner")
	}
	if !strings.Contains(body, `hx-target="#alert-banner"`) {
		t.Fatalf("expected dismiss hx-target to alert banner")
	}
	if !strings.Contains(body, `hx-swap="outerHTML"`) {
		t.Fatalf("expected dismiss hx-swap outerHTML")
	}
}

func TestAlertDismissHidesBannerOnSubsequentHomeLoad(t *testing.T) {
	if activeAlertVersion == "" {
		t.Skip("no active alert configured")
	}

	s := newHomeEntriesHTTPTestServer(t)

	dismissReq := authedRequest(t, s, http.MethodPost, "/alert/"+activeAlertVersion+"/dismiss", strings.NewReader(""))
	dismissReq.Header.Set("HX-Request", "true")
	dismissRR := httptest.NewRecorder()
	s.ServeHTTP(dismissRR, dismissReq)

	if dismissRR.Code != http.StatusOK {
		t.Fatalf("expected 200 on dismiss, got %d", dismissRR.Code)
	}

	homeReq := authedRequest(t, s, http.MethodGet, "/", nil)
	homeRR := httptest.NewRecorder()
	s.ServeHTTP(homeRR, homeReq)

	if homeRR.Code != http.StatusOK {
		t.Fatalf("expected 200 on home, got %d", homeRR.Code)
	}
	if strings.Contains(homeRR.Body.String(), `id="alert-banner"`) {
		t.Fatalf("expected alert banner to be hidden after dismiss")
	}
}
