package server

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

type entriesAPIResponse struct {
	Entries     []entryAPIItem `json:"entries"`
	SelectedDay string         `json:"selected_day"`
}

type entryAPIItem struct {
	ID                  int64                   `json:"id"`
	RecordedAtUtc       int64                   `json:"recorded_at_utc"`
	EntryDate           string                  `json:"entry_date"`
	NoteText            *string                 `json:"note_text"`
	IsPrivate           bool                    `json:"is_private"`
	IsDraft             bool                    `json:"is_draft"`
	AudioFilePath       *string                 `json:"audio_file_path"`
	TranscriptionStatus string                  `json:"transcription_status"`
	Trackables          []entryTrackableAPIItem `json:"trackables"`
}

type entryTrackableAPIItem struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Value string `json:"value"`
}

func nullableEntryString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func mapEntryViewsToAPIItems(entries []entryView) []entryAPIItem {
	items := make([]entryAPIItem, 0, len(entries))
	for _, entry := range entries {
		trackables := make([]entryTrackableAPIItem, 0, len(entry.Trackables))
		for _, trackable := range entry.Trackables {
			trackables = append(trackables, entryTrackableAPIItem{
				Name:  trackable.Name,
				Icon:  trackable.Icon,
				Value: trackable.Value,
			})
		}

		items = append(items, entryAPIItem{
			ID:                  entry.ID,
			RecordedAtUtc:       entry.RecordedAtUtc,
			EntryDate:           entry.EntryDate,
			NoteText:            nullableEntryString(entry.NoteText),
			IsPrivate:           entry.IsPrivate == 1,
			IsDraft:             entry.IsDraft,
			AudioFilePath:       nullableEntryString(entry.AudioFilePath),
			TranscriptionStatus: entry.TranscriptionStatus,
			Trackables:          trackables,
		})
	}

	return items
}

func (s *Server) entries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	now := time.Now()
	selectedDay, err := parseSelectedDay(r.URL.Query().Get("day"), now)
	if err != nil {
		log.Printf("Invalid day format: %v", err)
		respondBadRequest(w, r, "Invalid day format")
		return
	}

	selectedDayStr := selectedDay.Format(dateLayoutISO)
	todayStr := now.Format(dateLayoutISO)

	entries, err := s.listEntryViewsByDay(r.Context(), userID, selectedDayStr)
	if err != nil {
		log.Printf("Failed to list entries: %v", err)
		respondInternalError(w, r, "Failed to load entries")
		return
	}

	entryTrackableDialogData, err := s.buildTrackablePickerData(r.Context(), userID, 0, false, "entries-dialog", true)
	if err != nil {
		log.Printf("Failed to build trackable picker data: %v", err)
		respondInternalError(w, r, "Failed to load trackables")
		return
	}

	s.renderPage(w, r, "entries_title", "entries_content", map[string]interface{}{
		"Entries":                  entries,
		"SelectedDay":              selectedDayStr,
		"TodayStr":                 todayStr,
		"DayNavigation":            buildDayNavigation(selectedDay),
		"EntryTrackableDialogData": entryTrackableDialogData,
	})
}

func (s *Server) entriesAPI(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	now := time.Now()
	selectedDay, err := parseSelectedDay(r.URL.Query().Get("day"), now)
	if err != nil {
		log.Printf("Invalid day format: %v", err)
		respondBadRequest(w, r, "Invalid day format")
		return
	}

	selectedDayStr := selectedDay.Format(dateLayoutISO)

	entries, err := s.listEntryViewsByDay(r.Context(), userID, selectedDayStr)
	if err != nil {
		log.Printf("Failed to list entries for API: %v", err)
		respondInternalError(w, r, "Failed to load entries")
		return
	}

	respondJSON(w, http.StatusOK, entriesAPIResponse{
		Entries:     mapEntryViewsToAPIItems(entries),
		SelectedDay: selectedDayStr,
	})
}

func (s *Server) entryItem(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	entryID, ok := requirePathInt64(w, r, "id", "entry ID")
	if !ok {
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
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	entryID, ok := requirePathInt64(w, r, "id", "entry ID")
	if !ok {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	err := s.queries.DeleteEntry(r.Context(), db.DeleteEntryParams{
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
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		s.renderPage(w, r, "entries_add_title", "entries_add_content", nil)
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if !requireParsedForm(w, r) {
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
	if s.devMode {
		time.Sleep(devAddEntryDelay)
	}
	req := classifyRequest(r)
	if req.IsHTMX {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/entries", http.StatusSeeOther)
}
