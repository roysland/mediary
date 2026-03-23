package server

import (
	"net/http"
	"strings"

	"roysland.me/symptomstracker/internal/auth"
)

func withSessionRequired(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !routeExists(mux, r) {
			respondNotFound(w, r, "Not found")
			return
		}

		if isPublicRoute(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}

		user := auth.CurrentUser(r)
		if user == nil || user.ID <= 0 {
			req := classifyRequest(r)
			if req.IsAJAX || req.AcceptsJSON {
				respondUnauthorized(w, r)
				return
			}
			// For normal browser GET requests, redirect to the auth page.
			// For API/HTMX/XHR/JSON requests (or non-GET methods), respond with 401.
			isHTMX := r.Header.Get("HX-Request") == "true"
			isXHR := r.Header.Get("X-Requested-With") == "XMLHttpRequest"
			acceptsJSON := strings.Contains(r.Header.Get("Accept"), "application/json")

			if r.Method != http.MethodGet || isHTMX || isXHR || acceptsJSON {
				respondUnauthorized(w, r)
				return
			}

			http.Redirect(w, r, "/auth", http.StatusSeeOther)
			return
		}

		_ = auth.RefreshCurrentSession(w, r)

		mux.ServeHTTP(w, r)
	})
}

func routeExists(mux *http.ServeMux, r *http.Request) bool {
	_, pattern := mux.Handler(r)
	return pattern != ""
}

func isPublicRoute(path string) bool {
	switch {
	case path == "/healthz":
		return true
	case path == "/auth":
		return true
	case isPublicE2ERoute(path):
		return true
	case path == "/auth/logout":
		return true
	case path == "/webauthn/login/options":
		return true
	case path == "/webauthn/login/verify":
		return true
	case path == "/webauthn/register/options":
		return true
	case path == "/webauthn/register/verify":
		return true
	case path == "/webauthn/passkeys/options":
		return true
	case path == "/webauthn/passkeys/verify":
		return true
	case path == "/auth/passkeys/register/options":
		return true
	case path == "/auth/passkeys/register/verify":
		return true
	case path == "/link":
		return true
	case path == "/static/", path == "/favicon.ico":
		return true
	case len(path) >= len("/static/") && path[:len("/static/")] == "/static/":
		return true
	default:
		return false
	}
}
