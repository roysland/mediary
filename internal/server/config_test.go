package server

import (
	"reflect"
	"testing"
)

func TestGetEnvCSV(t *testing.T) {
	t.Setenv("CSRF_TRUSTED_ORIGINS", " https://app.example.com,https://staging.example.com ,, ")

	got := getEnvCSV("CSRF_TRUSTED_ORIGINS")
	want := []string{"https://app.example.com", "https://staging.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected csv values: got %#v want %#v", got, want)
	}
}

func TestLoadConfigReadsTrustedOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CSRF_TRUSTED_ORIGINS", "https://app.example.com,https://cdn.example.com")
	t.Setenv("AUTH_SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("WEBAUTHN_RP_ID", "app.example.com")
	t.Setenv("WEBAUTHN_RP_DISPLAY_NAME", "My Diary")
	t.Setenv("WEBAUTHN_RP_ORIGINS", "https://app.example.com,https://m.app.example.com")

	cfg := LoadConfig()
	want := []string{"https://app.example.com", "https://cdn.example.com"}
	if !reflect.DeepEqual(cfg.CSRFTrustedOrigins, want) {
		t.Fatalf("unexpected CSRFTrustedOrigins: got %#v want %#v", cfg.CSRFTrustedOrigins, want)
	}
	if cfg.AuthSessionSecret != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected AuthSessionSecret: %q", cfg.AuthSessionSecret)
	}
	if cfg.WebAuthnRPID != "app.example.com" {
		t.Fatalf("unexpected WebAuthnRPID: %q", cfg.WebAuthnRPID)
	}
	if cfg.WebAuthnRPDisplayName != "My Diary" {
		t.Fatalf("unexpected WebAuthnRPDisplayName: %q", cfg.WebAuthnRPDisplayName)
	}
	rpOrigins := []string{"https://app.example.com", "https://m.app.example.com"}
	if !reflect.DeepEqual(cfg.WebAuthnRPOrigins, rpOrigins) {
		t.Fatalf("unexpected WebAuthnRPOrigins: got %#v want %#v", cfg.WebAuthnRPOrigins, rpOrigins)
	}
}

func TestLoadConfigDefaultWebAuthnOrigins(t *testing.T) {
	t.Setenv("WEBAUTHN_RP_ORIGINS", "")

	cfg := LoadConfig()
	want := []string{"http://localhost:8080"}
	if !reflect.DeepEqual(cfg.WebAuthnRPOrigins, want) {
		t.Fatalf("unexpected default WebAuthnRPOrigins: got %#v want %#v", cfg.WebAuthnRPOrigins, want)
	}
}

func TestValidateWebAuthnConfig_AcceptsMatchingOrigins(t *testing.T) {
	cfg := Config{
		DevMode:           false,
		WebAuthnRPID:      "airberry.no",
		WebAuthnRPOrigins: []string{"https://diary.airberry.no", "https://airberry.no"},
	}

	if err := validateWebAuthnConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateWebAuthnConfig_RejectsMismatchedHost(t *testing.T) {
	cfg := Config{
		DevMode:           false,
		WebAuthnRPID:      "airberry.no",
		WebAuthnRPOrigins: []string{"https://example.com"},
	}

	err := validateWebAuthnConfig(cfg)
	if err == nil {
		t.Fatal("expected mismatched host validation error, got nil")
	}
}

func TestValidateWebAuthnConfig_RejectsHTTPInProduction(t *testing.T) {
	cfg := Config{
		DevMode:           false,
		WebAuthnRPID:      "localhost",
		WebAuthnRPOrigins: []string{"http://localhost:8080"},
	}

	err := validateWebAuthnConfig(cfg)
	if err == nil {
		t.Fatal("expected http origin validation error in production, got nil")
	}
}
