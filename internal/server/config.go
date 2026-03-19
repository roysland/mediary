package server

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds the server configuration loaded from environment variables.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string
	// ListenAddr is the address and port to listen on (e.g., ":8080").
	ListenAddr string
	// DevMode indicates whether the server is running in development mode.
	DevMode bool

	// AudioStorageDir is the directory where uploaded voice recordings are stored.
	AudioStorageDir string
	// WhisperBinaryPath is the path (or name on PATH) of the whisper.cpp binary.
	// Leave empty to disable transcription.
	WhisperBinaryPath string
	// WhisperModelPath is the path to the ggml model file used by whisper.cpp.
	WhisperModelPath string
	// FFmpegBinaryPath is the path (or name on PATH) of ffmpeg, used to convert
	// browser audio to the WAV format that whisper.cpp requires.
	FFmpegBinaryPath string
	// TranscriptionTimeoutSeconds is the maximum time allowed for a single
	// transcription job (ffmpeg + whisper). Defaults to 120.
	TranscriptionTimeoutSeconds int

	// CSRFTrustedOrigins lists additional origins allowed for cross-origin write
	// requests when using net/http CrossOriginProtection.
	CSRFTrustedOrigins []string

	// AuthSessionSecret signs auth cookies. Must be set in production.
	AuthSessionSecret string

	// WebAuthnRPID is the relying party ID for passkey ceremonies.
	WebAuthnRPID string
	// WebAuthnRPDisplayName is shown to users during passkey flows.
	WebAuthnRPDisplayName string
	// WebAuthnRPOrigins are allowed origins for WebAuthn ceremonies.
	WebAuthnRPOrigins []string
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig() Config {
	cfg := Config{
		DBPath:                      getEnv("DB_PATH", "data/app.db"),
		ListenAddr:                  getEnv("LISTEN_ADDR", ":8080"),
		DevMode:                     os.Getenv("APP_ENV") != "production",
		AudioStorageDir:             getEnv("AUDIO_STORAGE_DIR", "data/audio"),
		WhisperBinaryPath:           getEnv("WHISPER_BINARY_PATH", ""),
		WhisperModelPath:            getEnv("WHISPER_MODEL_PATH", ""),
		FFmpegBinaryPath:            getEnv("FFMPEG_BINARY_PATH", "ffmpeg"),
		TranscriptionTimeoutSeconds: getEnvInt("TRANSCRIPTION_TIMEOUT_SECONDS", 120),
		CSRFTrustedOrigins:          getEnvCSV("CSRF_TRUSTED_ORIGINS"),
		AuthSessionSecret:           getEnv("AUTH_SESSION_SECRET", ""),
		WebAuthnRPID:                getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnRPDisplayName:       getEnv("WEBAUTHN_RP_DISPLAY_NAME", "Symptoms Tracker"),
		WebAuthnRPOrigins:           getEnvCSVWithDefault("WEBAUTHN_RP_ORIGINS", []string{"http://localhost:8080"}),
	}
	return cfg
}

// getEnv returns the value of an environment variable, or a default if not set.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns an environment variable parsed as int, or a default if not set or invalid.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvCSV splits a comma-separated environment variable into trimmed values.
// Empty items are discarded.
func getEnvCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

func getEnvCSVWithDefault(key string, fallback []string) []string {
	values := getEnvCSV(key)
	if len(values) == 0 {
		return fallback
	}

	return values
}

func validateWebAuthnConfig(cfg Config) error {
	rpID := strings.TrimSpace(cfg.WebAuthnRPID)
	if rpID == "" {
		return fmt.Errorf("WEBAUTHN_RP_ID must not be empty")
	}

	if len(cfg.WebAuthnRPOrigins) == 0 {
		return fmt.Errorf("WEBAUTHN_RP_ORIGINS must include at least one origin")
	}

	for _, rawOrigin := range cfg.WebAuthnRPOrigins {
		origin := strings.TrimSpace(rawOrigin)
		if origin == "" {
			continue
		}

		parsed, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid WEBAUTHN_RP_ORIGINS value %q: %w", origin, err)
		}

		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS value %q must include scheme and host", origin)
		}

		if parsed.Path != "" && parsed.Path != "/" {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS value %q must not include a path", origin)
		}

		if parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS value %q must not include query params or fragments", origin)
		}

		hostname := strings.TrimSpace(parsed.Hostname())
		if hostname == "" {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS value %q must include a valid host", origin)
		}

		// Per WebAuthn requirements, RP ID must equal the origin hostname or be its registrable suffix.
		if hostname != rpID && !strings.HasSuffix(hostname, "."+rpID) {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS host %q does not match WEBAUTHN_RP_ID %q", hostname, rpID)
		}

		if !cfg.DevMode && parsed.Scheme != "https" {
			return fmt.Errorf("WEBAUTHN_RP_ORIGINS value %q must use https in production", origin)
		}
	}

	return nil
}
