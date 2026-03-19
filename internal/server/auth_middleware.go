package server

import (
	"net/http"

	"roysland.me/symptomstracker/internal/auth"
)

func withSessionRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user := auth.CurrentUser(r)
		if user == nil || user.ID <= 0 {
			req := classifyRequest(r)
			if req.IsAJAX || req.AcceptsJSON {
				respondUnauthorized(w, r)
				return
			}
			http.Redirect(w, r, "/auth", http.StatusSeeOther)
			return
		}

		_ = auth.RefreshCurrentSession(w, r)

		next.ServeHTTP(w, r)
	})
}

func isPublicRoute(path string) bool {
	switch {
	case path == "/auth":
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
	case path == "/static/", path == "/favicon.ico":
		return true
	case len(path) >= len("/static/") && path[:len("/static/")] == "/static/":
		return true
	default:
		return false
	}
}
