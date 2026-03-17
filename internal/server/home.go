package server

import (
	"log"
	"net/http"
	"time"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	today := time.Now().Format("2006-01-02")
	entries, err := s.listEntryViewsByDay(r.Context(), userID, today)
	if err != nil {
		log.Printf("Failed to list today's entries for home: %v", err)
		respondInternalError(w, r, "Failed to load home")
		return
	}

	s.renderPage(w, r, "home_title", "home_content", map[string]interface{}{
		"Entries":     entries,
		"SelectedDay": today,
		"TodayStr":    today,
	})
}
