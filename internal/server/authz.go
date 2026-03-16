package server

import (
	"net/http"

	"roysland.me/symptomstracker/internal/auth"
)

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	user := auth.CurrentUser(r)
	if user == nil || user.ID <= 0 {
		respondUnauthorized(w, r)
		return 0, false
	}

	return user.ID, true
}
