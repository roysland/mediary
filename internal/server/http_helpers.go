package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func requireMethod(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, method := range allowed {
		if r.Method == method {
			return true
		}
	}

	w.Header().Set("Allow", strings.Join(allowed, ", "))
	respondMethodNotAllowed(w, r)
	return false
}

// safeRedirect performs an HTTP redirect to a URL, but only if the URL is a
// relative path (no scheme, no host). This prevents open redirect vulnerabilities.
func safeRedirect(w http.ResponseWriter, r *http.Request, urlPath string, statusCode int) {
	// Trim whitespace
	urlPath = strings.TrimSpace(urlPath)

	// Default to home if empty
	if urlPath == "" {
		urlPath = "/"
	}

	// Parse the URL to check for unsafe redirects
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		// If parsing fails, use safe default
		http.Redirect(w, r, "/", statusCode)
		return
	}

	// Reject if URL has a scheme (http://, https://, //, etc.)
	if parsedURL.Scheme != "" {
		http.Redirect(w, r, "/", statusCode)
		return
	}

	// Reject if URL has a host (would indicate an absolute URL to another domain)
	if parsedURL.Host != "" {
		http.Redirect(w, r, "/", statusCode)
		return
	}

	// Reject if URL contains :// or @ (common open redirect patterns)
	if strings.Contains(urlPath, "://") || strings.Contains(urlPath, "@") {
		http.Redirect(w, r, "/", statusCode)
		return
	}

	// Additional check: if it looks like a network address, reject it
	if net.ParseIP(urlPath) != nil {
		http.Redirect(w, r, "/", statusCode)
		return
	}

	// Safe to redirect to relative path
	//nolint:gosec // G710: URL is validated above to be safe (no scheme, host, or suspicious patterns)
	http.Redirect(w, r, urlPath, statusCode)
}

func requirePathInt64(w http.ResponseWriter, r *http.Request, paramName, label string) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue(paramName))
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		respondBadRequest(w, r, fmt.Sprintf("Invalid %s", label))
		return 0, false
	}

	return value, true
}

func requireParsedForm(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		respondBadRequest(w, r, "Invalid form data")
		return false
	}

	return true
}
