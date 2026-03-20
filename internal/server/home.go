package server

import (
	"net/http"
	"time"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	_, ok := requireUserID(w, r)
	if !ok {
		return
	}

	today := time.Now().Format(dateLayoutISO)
	s.renderPage(w, r, "home_title", "home_content", map[string]interface{}{
		"SelectedDay": today,
		"TodayStr":    today,
		"AddEntryForm": buildEntryFormViewData(
			"/entry/add",
			today,
			today,
			false,
			true,
			false,
		),
	})
}
