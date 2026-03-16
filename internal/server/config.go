package server

import (
	"os"
)

// Config holds the server configuration loaded from environment variables.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string
	// ListenAddr is the address and port to listen on (e.g., ":8080").
	ListenAddr string
	// DevMode indicates whether the server is running in development mode.
	DevMode bool
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig() Config {
	cfg := Config{
		DBPath:     getEnv("DB_PATH", "data/app.db"),
		ListenAddr: getEnv("LISTEN_ADDR", ":8080"),
		DevMode:    os.Getenv("APP_ENV") != "production",
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
