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
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if r.Method == http.MethodGet {
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
		return
	}

	if !requireParsedForm(w, r) {
		return
	}

	formValueWithFallback := func(primary string, fallbacks ...string) string {
		value := strings.TrimSpace(r.FormValue(primary))
		if value != "" {
			return value
		}
		for _, fallback := range fallbacks {
			value = strings.TrimSpace(r.FormValue(fallback))
			if value != "" {
				return value
			}
		}
		return ""
	}

	now := time.Now()
	name, err := requireNonEmpty(formValueWithFallback("trackable_name"), "trackable_name")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	valueTypeRaw := formValueWithFallback("trackable_value_type", "trackable_value-type")
	valueType, err := requireOneOf(valueTypeRaw, "trackable_value_type", "integer", "boolean", "text")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	icon := formValueWithFallback("trackable_icon")
	unit := formValueWithFallback("trackable_unit")
	privateLabel := formValueWithFallback("trackable_private_label", "trackable_private-label", "trackable_description")

	toNullString := func(s string) sql.NullString {
		if s == "" {
			return sql.NullString{}
		}
		return sql.NullString{String: s, Valid: true}
	}

	iconVal := toNullString(icon)
	unitVal := toNullString(unit)
	privateLabelVal := toNullString(privateLabel)

	isSensitiveRaw := formValueWithFallback("trackable_is_sensitive", "trackable_is-sensitive")
	isSensitive, err := checkboxToInt64(isSensitiveRaw, "trackable_is_sensitive")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	minVal, err := optionalInt64(formValueWithFallback("trackable_min_value"), "trackable_min_value")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	maxVal, err := optionalInt64(formValueWithFallback("trackable_max_value"), "trackable_max_value")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	if minVal.Valid && maxVal.Valid && minVal.Int64 > maxVal.Int64 {
		respondBadRequest(w, r, "trackable_min_value must be less than or equal to trackable_max_value")
		return
	}

	templateIDRaw := formValueWithFallback("trackable_template_id", "presetId")
	templateID, err := optionalInt64(templateIDRaw, "trackable_template_id")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}
	if templateID.Valid {
		if templateID.Int64 <= 0 {
			respondBadRequest(w, r, "trackable_template_id must be positive")
			return
		}

		_, err = s.queries.GetAvailableTrackableTemplateByID(r.Context(), db.GetAvailableTrackableTemplateByIDParams{
			UserID: userID,
			ID:     templateID.Int64,
		})
		if err == sql.ErrNoRows {
			respondBadRequest(w, r, "Invalid or unavailable trackable preset")
			return
		}
		if err != nil {
			respondInternalError(w, r, "Failed to validate trackable preset")
			return
		}
	}

	category := formValueWithFallback("trackable_category")
	if category == "" {
		category = "default"
	}
	category, err = requireOneOf(category, "trackable_category", "default", "symptom", "activity", "measurement", "state")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	if valueType != "integer" {
		// Integer-only fields should not be persisted for non-integer trackables.
		minVal = sql.NullInt64{}
		maxVal = sql.NullInt64{}
		unit = ""
		unitVal = sql.NullString{}
	}

	_, err = s.queries.CreateTrackableDefinition(r.Context(), db.CreateTrackableDefinitionParams{
		UserID:       userID,
		TemplateID:   templateID,
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

}

func (s *Server) listTrackables(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

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

	_, err := s.queries.GetEntryByID(r.Context(), db.GetEntryByIDParams{
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
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	id, ok := requirePathInt64(w, r, "id", "trackable ID")
	if !ok {
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
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	trackableID, ok := requirePathInt64(w, r, "id", "trackable ID")
	if !ok {
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
		if !requireParsedForm(w, r) {
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
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	trackableID, ok := requirePathInt64(w, r, "id", "trackable ID")
	if !ok {
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
		if !requireParsedForm(w, r) {
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
