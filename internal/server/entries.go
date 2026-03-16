package server

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

func (s *Server) entries(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	selectedDay, err := parseSelectedDay(r.URL.Query().Get("day"), now)
	if err != nil {
		log.Printf("Invalid day format: %v", err)
		respondBadRequest(w, r, "Invalid day format")
		return
	}

	selectedDayStr := selectedDay.Format("2006-01-02")
	todayStr := now.Format("2006-01-02")

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	entries, err := s.listEntryViewsByDay(r.Context(), userID, selectedDayStr)
	if err != nil {
		log.Printf("Failed to list entries: %v", err)
		respondInternalError(w, r, "Failed to load entries")
		return
	}

	// Get all tracked trackables
	trackables, err := s.queries.ListTrackableDefinitions(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to list trackables: %v", err)
		respondInternalError(w, r, "Failed to load trackables")
		return
	}
	trackableMap := make(map[int64]db.TrackableDefinition)
	for _, t := range trackables {
		trackableMap[t.ID] = t
	}

	s.renderPage(w, r, "entries_title", "entries_content", map[string]interface{}{
		"Entries":       entries,
		"Trackables":    trackableMap,
		"SelectedDay":   selectedDayStr,
		"TodayStr":      todayStr,
		"DayNavigation": buildDayNavigation(selectedDay),
	})
}

func (s *Server) entryItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondMethodNotAllowed(w, r)
		return
	}

	entryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || entryID <= 0 {
		respondBadRequest(w, r, "Invalid entry ID")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	entry, err := s.loadEntryViewByID(r.Context(), userID, entryID)
	if errors.Is(err, errEntryNotFound) {
		respondNotFound(w, r, "Entry not found")
		return
	}
	if err != nil {
		log.Printf("Failed to fetch entry %d: %v", entryID, err)
		respondInternalError(w, r, "Failed to load entry")
		return
	}

	s.renderTemplate(w, r, "entry_item", entry)
}

func (s *Server) deleteEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondMethodNotAllowed(w, r)
		return
	}

	entryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || entryID <= 0 {
		respondBadRequest(w, r, "Invalid entry ID")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	err = s.queries.DeleteEntry(r.Context(), db.DeleteEntryParams{
		ID:     entryID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("Failed to delete entry %d: %v", entryID, err)
		respondInternalError(w, r, "Failed to delete entry")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) addEntry(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderPage(w, r, "entries_add_title", "entries_add_content", nil)
	case http.MethodPost:
		userID, ok := requireUserID(w, r)
		if !ok {
			return
		}

		if err := r.ParseForm(); err != nil {
			respondBadRequest(w, r, "Invalid form data")
			return
		}

		now := time.Now()
		note, err := requireNonEmpty(r.FormValue("entry_input"), "entry_input")
		if err != nil {
			respondBadRequest(w, r, err.Error())
			return
		}

		isPrivate, err := checkboxToInt64(r.FormValue("is_private_entry"), "is_private_entry")
		if err != nil {
			respondBadRequest(w, r, err.Error())
			return
		}
		noteText := sql.NullString{String: note, Valid: note != ""}

		entry, err := s.createEntry(r.Context(), userID, now, noteText, isPrivate)
		if err != nil {
			log.Printf("Failed to create entry: %v", err)
			respondInternalError(w, r, "Failed to save entry")
			return
		}
		log.Printf("Created entry: %+v", entry)
		time.Sleep(500 * time.Millisecond)
		req := classifyRequest(r)
		if req.IsHTMX {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/entries", http.StatusSeeOther)
	default:
		respondMethodNotAllowed(w, r)

	}
}
