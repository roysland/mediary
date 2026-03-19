package server

import (
	"net/http"

	"roysland.me/symptomstracker/internal/auth"
)

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	user := auth.CurrentUser(r)
	if user == nil || user.ID <= 0 {
		req := classifyRequest(r)
		if !req.IsAJAX && !req.AcceptsJSON && r.Method == http.MethodGet {
			http.Redirect(w, r, "/auth", http.StatusSeeOther)
			return 0, false
		}
		respondUnauthorized(w, r)
		return 0, false
	}

	return user.ID, true
}
