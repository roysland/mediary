package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifyRequest(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		wantIsAJAX  bool
		wantIsJSON  bool
		wantIsHTMX  bool
		wantIsXHR   bool
		wantAccepts bool
	}{
		{
			name:       "plain request",
			headers:    map[string]string{},
			wantIsAJAX: false,
		},
		{
			name: "json content type",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantIsAJAX: true,
			wantIsJSON: true,
		},
		{
			name: "htmx request",
			headers: map[string]string{
				"HX-Request": "true",
			},
			wantIsAJAX: true,
			wantIsHTMX: true,
		},
		{
			name: "xhr with accept json",
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
				"Accept":           "text/html,application/json",
			},
			wantIsAJAX:  true,
			wantIsXHR:   true,
			wantAccepts: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := classifyRequest(req)
			if got.IsAJAX != tt.wantIsAJAX || got.IsJSON != tt.wantIsJSON || got.IsHTMX != tt.wantIsHTMX || got.IsXHR != tt.wantIsXHR || got.AcceptsJSON != tt.wantAccepts {
				t.Fatalf("unexpected classification: %+v", got)
			}
		})
	}
}

func TestRequireNonEmpty(t *testing.T) {
	got, err := requireNonEmpty("  hello  ", "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected trimmed value, got %q", got)
	}

	_, err = requireNonEmpty("   ", "field")
	if err == nil {
		t.Fatal("expected error for empty value")
	}
}

func TestOptionalInt64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		valid   bool
		value   int64
		wantErr bool
	}{
		{name: "empty", input: "", valid: false, value: 0},
		{name: "integer", input: "42", valid: true, value: 42},
		{name: "invalid", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optionalInt64(tt.input, "age")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Valid != tt.valid || got.Int64 != tt.value {
				t.Fatalf("unexpected value: %+v", got)
			}
		})
	}
}

func TestRequireOneOf(t *testing.T) {
	got, err := requireOneOf(" dark ", "theme", "light", "dark", "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dark" {
		t.Fatalf("expected dark, got %q", got)
	}

	_, err = requireOneOf("blue", "theme", "light", "dark", "system")
	if err == nil {
		t.Fatal("expected error for invalid option")
	}
}

func TestCheckboxToInt64(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{input: "", want: 0},
		{input: "on", want: 1},
		{input: "true", want: 1},
		{input: "1", want: 1},
		{input: "off", want: 0},
		{input: "false", want: 0},
		{input: "0", want: 0},
		{input: "wat", wantErr: true},
	}

	for _, tt := range tests {
		got, err := checkboxToInt64(tt.input, "private")
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for input %q", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("expected %d for input %q, got %d", tt.want, tt.input, got)
		}
	}
}

func TestRespondErrorJSONForAjax(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	respondBadRequest(rr, req, "bad input")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected json content type, got %q", ct)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected response body")
	}
}

func TestOptionalInt64ZeroValueIsNull(t *testing.T) {
	got, err := optionalInt64("   ", "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (sql.NullInt64{}) {
		t.Fatalf("expected zero null int64, got %+v", got)
	}
}

func TestWithServiceWorkerAllowedSetsHeader(t *testing.T) {
	h := withServiceWorkerAllowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/sw.js?v=dev", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Service-Worker-Allowed"); got != "/" {
		t.Fatalf("expected Service-Worker-Allowed=/, got %q", got)
	}
}
