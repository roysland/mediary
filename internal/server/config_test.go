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

	cfg := LoadConfig()
	want := []string{"https://app.example.com", "https://cdn.example.com"}
	if !reflect.DeepEqual(cfg.CSRFTrustedOrigins, want) {
		t.Fatalf("unexpected CSRFTrustedOrigins: got %#v want %#v", cfg.CSRFTrustedOrigins, want)
	}
}
