package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

type trackablePickerViewData struct {
	EntryID              int64
	HasEntryID           bool
	PickerID             string
	ShowAddTrackableLink bool
	ActiveTrackables     []db.ListTrackableDefinitionsWithDismissalRow
	DismissedTrackables  []db.ListTrackableDefinitionsWithDismissalRow
}

func (s *Server) addTrackable(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		trackableTemplate, err := s.queries.GetTrackableTemplates(r.Context(), userID)
		if err != nil {
			respondInternalError(w, r, "Failed to fetch trackable templates")
			return
		}

		data := struct {
			TrackablePresets []db.TrackableTemplate
		}{
			TrackablePresets: trackableTemplate,
		}
		s.renderPage(w, r, "trackable_add_title", "trackable_add_content", data)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			respondBadRequest(w, r, "Invalid form data")
			return
		}

		now := time.Now()
		name := strings.TrimSpace(r.FormValue("trackable_name"))
		valueType := strings.TrimSpace(r.FormValue("trackable_value-type"))
		icon := strings.TrimSpace(r.FormValue("trackable_icon"))
		unit := strings.TrimSpace(r.FormValue("trackable_unit"))
		minValue := strings.TrimSpace(r.FormValue("trackable_min_value"))
		maxValue := strings.TrimSpace(r.FormValue("trackable_max_value"))
		privateLabel := strings.TrimSpace(r.FormValue("trackable_private-label"))

		toNullString := func(s string) sql.NullString {
			if s == "" {
				return sql.NullString{}
			}
			return sql.NullString{String: s, Valid: true}
		}

		iconVal := toNullString(icon)
		unitVal := toNullString(unit)
		privateLabelVal := toNullString(privateLabel)

		isSensitive := int64(0)
		if r.FormValue("trackable_is-sensitive") == "on" {
			isSensitive = 1
		}

		minVal := sql.NullInt64{}
		if minValue != "" {
			if v, err := strconv.ParseInt(minValue, 10, 64); err == nil {
				minVal = sql.NullInt64{Int64: v, Valid: true}
			}
		}

		maxVal := sql.NullInt64{}
		if maxValue != "" {
			if v, err := strconv.ParseInt(maxValue, 10, 64); err == nil {
				maxVal = sql.NullInt64{Int64: v, Valid: true}
			}
		}

		category := strings.TrimSpace(r.FormValue("trackable_category"))
		if category == "" {
			category = "default"
		}

		_, err := s.queries.CreateTrackableDefinition(r.Context(), db.CreateTrackableDefinitionParams{
			UserID:       userID,
			Name:         name,
			ValueType:    valueType,
			Icon:         iconVal,
			Unit:         unitVal,
			MinValue:     minVal,
			MaxValue:     maxVal,
			IsSensitive:  isSensitive,
			Category:     category,
			PrivateLabel: privateLabelVal,
			CreatedAtUtc: now.UTC().Unix(),
		})
		if err != nil {
			fmt.Printf("Failed to create trackable definition: %v\n", err)
			respondInternalError(w, r, "Failed to create trackable")
			return
		}

		http.Redirect(w, r, "/trackables", http.StatusSeeOther)
	default:
		respondMethodNotAllowed(w, r)
	}

}

func (s *Server) listTrackables(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	data, err := s.buildTrackablePickerData(r.Context(), userID, 0, false, "trackable-page", true)
	if err != nil {
		respondInternalError(w, r, "Failed to fetch trackables")
		return
	}
	s.renderPage(w, r, "trackable_list_title", "trackable_list_content", data)
}

func (s *Server) entryTrackablesDialog(w http.ResponseWriter, r *http.Request) {
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

	_, err = s.queries.GetEntryByID(r.Context(), db.GetEntryByIDParams{
		ID:     entryID,
		UserID: userID,
	})
	if err == sql.ErrNoRows {
		respondNotFound(w, r, "Entry not found")
		return
	}
	if err != nil {
		respondInternalError(w, r, "Failed to fetch entry")
		return
	}

	data, err := s.buildTrackablePickerData(r.Context(), userID, entryID, true, fmt.Sprintf("entry-%d", entryID), false)
	if err != nil {
		respondInternalError(w, r, "Failed to fetch trackables")
		return
	}

	s.renderTemplate(w, r, "entry_trackable_dialog_content", data)
}

func (s *Server) registerTrackable(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	// Extract trackable ID from URL
	idStr := strings.TrimPrefix(r.URL.Path, "/trackable/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondBadRequest(w, r, "Invalid trackable ID")
		return
	}

	trackable, err := s.queries.GetTrackableById(r.Context(), db.GetTrackableByIdParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondNotFound(w, r, "Trackable not found")
		} else {
			respondInternalError(w, r, "Failed to fetch trackable")
		}
		return
	}

	s.renderPage(w, r, "register_trackable_title", "register_trackable_content", trackable)
}

func (s *Server) saveTrackableValue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondMethodNotAllowed(w, r)
		return
	}

	idStr := r.PathValue("id")
	trackableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || trackableID <= 0 {
		respondBadRequest(w, r, "Invalid trackable ID")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	trackable, err := s.queries.GetTrackableById(r.Context(), db.GetTrackableByIdParams{ID: trackableID, UserID: userID})
	if err == sql.ErrNoRows {
		respondNotFound(w, r, "Trackable not found")
		return
	}
	if err != nil {
		respondInternalError(w, r, "Failed to fetch trackable")
		return
	}

	req := classifyRequest(r)

	var rawValue interface{}
	if req.IsJSON {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondBadRequest(w, r, "Invalid JSON body")
			return
		}
		rawValue = body["value"]
	} else {
		if err := r.ParseForm(); err != nil {
			respondBadRequest(w, r, "Invalid form data")
			return
		}
	}

	var valueInt sql.NullInt64
	var valueBool sql.NullInt64
	var valueText sql.NullString
	entryID := int64(0)
	hasEntryID := false
	if !req.IsJSON {
		if rawEntryID := strings.TrimSpace(r.FormValue("entry_id")); rawEntryID != "" {
			parsedEntryID, parseErr := strconv.ParseInt(rawEntryID, 10, 64)
			if parseErr != nil || parsedEntryID <= 0 {
				respondBadRequest(w, r, "Invalid entry ID")
				return
			}
			entryID = parsedEntryID
			hasEntryID = true
		}
	}

	switch trackable.ValueType {
	case "integer":
		var intVal int64
		if req.IsJSON {
			f, ok := rawValue.(float64)
			if !ok {
				respondBadRequest(w, r, "value must be a number")
				return
			}
			intVal = int64(f)
		} else {
			intVal, err = strconv.ParseInt(r.FormValue("value_int"), 10, 64)
			if err != nil {
				respondBadRequest(w, r, "value must be an integer")
				return
			}
		}
		valueInt = sql.NullInt64{Int64: intVal, Valid: true}
	case "boolean":
		valueBool = sql.NullInt64{Int64: 1, Valid: true}
	case "text":
		var textVal string
		if req.IsJSON {
			s, ok := rawValue.(string)
			if !ok {
				respondBadRequest(w, r, "value must be a string")
				return
			}
			textVal = s
		} else {
			textVal = r.FormValue("value_text")
		}
		if textVal == "" {
			respondBadRequest(w, r, "value must not be empty")
			return
		}
		valueText = sql.NullString{String: textVal, Valid: true}
	default:
		respondBadRequest(w, r, "Invalid trackable type")
		return
	}

	now := time.Now()
	result, err := s.saveTrackableValueForUser(r.Context(), userID, trackableSaveInput{
		TrackableID: trackableID,
		EntryID:     entryID,
		HasEntryID:  hasEntryID,
		ValueInt:    valueInt,
		ValueBool:   valueBool,
		ValueText:   valueText,
	}, now)
	if errors.Is(err, errTrackableNotFound) {
		respondNotFound(w, r, "Trackable not found")
		return
	}
	if errors.Is(err, errEntryNotFound) {
		respondNotFound(w, r, "Entry not found")
		return
	}
	if err != nil {
		respondInternalError(w, r, "Failed to save value")
		return
	}

	if req.IsAJAX {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"entry_id":  result.EntryID,
			"value_id":  result.ValueID,
			"timestamp": result.Timestamp,
		})
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/trackable"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func (s *Server) saveTrackableDismissal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondMethodNotAllowed(w, r)
		return
	}

	idStr := r.PathValue("id")
	trackableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || trackableID <= 0 {
		respondBadRequest(w, r, "Invalid trackable ID")
		return
	}

	req := classifyRequest(r)
	dismissed := true
	if req.IsJSON {
		var body struct {
			Dismissed *bool `json:"dismissed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondBadRequest(w, r, "Invalid JSON body")
			return
		}
		if body.Dismissed != nil {
			dismissed = *body.Dismissed
		}
	} else {
		if err := r.ParseForm(); err != nil {
			respondBadRequest(w, r, "Invalid form data")
			return
		}
		if raw := strings.TrimSpace(r.FormValue("dismissed")); raw != "" {
			parsed, err := strconv.ParseBool(raw)
			if err != nil {
				respondBadRequest(w, r, "dismissed must be a boolean")
				return
			}
			dismissed = parsed
		}
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	now := time.Now()
	state, err := s.saveTrackableDismissalForUser(r.Context(), userID, trackableID, dismissed, now)
	if errors.Is(err, errTrackableNotFound) {
		respondNotFound(w, r, "Trackable not found")
		return
	}
	if err != nil {
		respondInternalError(w, r, "Failed to save dismissal")
		return
	}

	if req.IsAJAX {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "success",
			"trackable_id": trackableID,
			"dismissed":    state.Dismissed == 1,
			"date":         state.DismissalDate,
		})
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/trackable"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}
