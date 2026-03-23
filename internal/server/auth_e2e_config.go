package server

import (
	"fmt"
	"strings"
)

func validateE2EAuthConfig(cfg Config) error {
	if strings.TrimSpace(cfg.E2EAuthToken) == "" {
		return nil
	}

	if cfg.AppEnv != "test" {
		return fmt.Errorf("PLAYWRIGHT_E2E_AUTH_TOKEN requires APP_ENV=test")
	}

	return nil
}
