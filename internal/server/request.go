package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type requestMeta struct {
	IsHTMX      bool
	IsJSON      bool
	IsXHR       bool
	AcceptsJSON bool
	IsAJAX      bool
}

func classifyRequest(r *http.Request) requestMeta {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	accept := strings.ToLower(r.Header.Get("Accept"))

	isJSON := strings.HasPrefix(contentType, "application/json")
	isHTMX := r.Header.Get("HX-Request") == "true"
	isXHR := r.Header.Get("X-Requested-With") == "XMLHttpRequest"
	acceptsJSON := strings.Contains(accept, "application/json")

	return requestMeta{
		IsHTMX:      isHTMX,
		IsJSON:      isJSON,
		IsXHR:       isXHR,
		AcceptsJSON: acceptsJSON,
		IsAJAX:      isJSON || isXHR || isHTMX,
	}
}

func requireNonEmpty(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return trimmed, nil
}

func optionalInt64(value, field string) (sql.NullInt64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullInt64{}, nil
	}

	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return sql.NullInt64{}, fmt.Errorf("%s must be an integer", field)
	}

	return sql.NullInt64{Int64: parsed, Valid: true}, nil
}

func requireOneOf(value, field string, allowed ...string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if slices.Contains(allowed, trimmed) {
		return trimmed, nil
	}

	return "", fmt.Errorf("invalid %s", field)
}

func checkboxToInt64(value, field string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}

	switch strings.ToLower(trimmed) {
	case "on", "true", "1":
		return 1, nil
	case "off", "false", "0":
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid %s", field)
	}
}
