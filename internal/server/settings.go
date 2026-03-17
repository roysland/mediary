package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

type UserSettings struct {
	Language   string
	Theme      string
	ScreenLock string
	ShareTimer string
}

type userDataExport struct {
	ExportedAtUTC            int64                        `json:"exported_at_utc"`
	UserID                   int64                        `json:"user_id"`
	Entries                  []exportEntry                `json:"entries"`
	TrackableValues          []exportTrackableValue       `json:"trackable_values"`
	TrackableDefinitions     []exportTrackableDefinition  `json:"trackable_definitions"`
	TrackableDailyDismissals []db.TrackableDailyDismissal `json:"trackable_daily_dismissals"`
	Settings                 map[string]*string           `json:"settings"`
	WebauthnCredentials      []exportWebauthnCredential   `json:"webauthn_credentials"`
}

type exportEntry struct {
	ID                    int64   `json:"id"`
	RecordedAtUTC         int64   `json:"recorded_at_utc"`
	TimezoneOffsetMinutes int64   `json:"timezone_offset_minutes"`
	EntryDate             string  `json:"entry_date"`
	NoteText              *string `json:"note_text"`
	IsPrivate             int64   `json:"is_private"`
	CreatedAtUTC          int64   `json:"created_at_utc"`
}

type exportTrackableValue struct {
	ID                    int64   `json:"id"`
	EntryID               int64   `json:"entry_id"`
	TrackableDefinitionID int64   `json:"trackable_definition_id"`
	ValueInt              *int64  `json:"value_int"`
	ValueBool             *int64  `json:"value_bool"`
	ValueText             *string `json:"value_text"`
	LocationText          *string `json:"location_text"`
	NoteText              *string `json:"note_text"`
	EntryDate             *string `json:"entry_date"`
	CreatedAtUTC          int64   `json:"created_at_utc"`
	UpdatedAtUTC          *int64  `json:"updated_at_utc"`
}

type exportTrackableDefinition struct {
	ID           int64   `json:"id"`
	TemplateID   *int64  `json:"template_id"`
	Name         string  `json:"name"`
	Icon         *string `json:"icon"`
	ValueType    string  `json:"value_type"`
	Unit         *string `json:"unit"`
	MinValue     *int64  `json:"min_value"`
	MaxValue     *int64  `json:"max_value"`
	IsSensitive  int64   `json:"is_sensitive"`
	PrivateLabel *string `json:"private_label"`
	Category     string  `json:"category"`
	Active       int64   `json:"active"`
	CreatedAtUTC int64   `json:"created_at_utc"`
}

type exportWebauthnCredential struct {
	ID           int64   `json:"id"`
	CredentialID string  `json:"credential_id"`
	PublicKey    string  `json:"public_key"`
	SignCount    int64   `json:"sign_count"`
	Transports   *string `json:"transports"`
	CreatedAtUTC int64   `json:"created_at_utc"`
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if r.Method == http.MethodGet {
		settings, err := s.loadUserSettings(r.Context(), userID)
		if err != nil {
			respondInternalError(w, r, "Failed to load settings")
			return
		}

		s.renderPage(w, r, "settings_title", "settings_content", map[string]any{
			"Settings": settings,
		})
		return
	}

	if !requireParsedForm(w, r) {
		return
	}

	language, err := requireOneOf(r.FormValue("language"), "language", "en", "no")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	theme, err := requireOneOf(r.FormValue("theme"), "theme", "light", "dark", "system")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	screenLock, err := requireOneOf(r.FormValue("screen_lock"), "screen_lock", "none", "60", "300", "600")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	shareTimer, err := requireOneOf(r.FormValue("share_timer"), "share_timer", "300", "600", "1800")
	if err != nil {
		respondBadRequest(w, r, err.Error())
		return
	}

	now := time.Now().UTC().Unix()
	settings := UserSettings{
		Language:   language,
		Theme:      theme,
		ScreenLock: screenLock,
		ShareTimer: shareTimer,
	}

	if err := s.saveUserSettings(r.Context(), userID, settings, now); err != nil {
		respondInternalError(w, r, "Failed to save settings")
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *Server) clearUserData(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if err := s.deleteAllUserData(r.Context(), userID); err != nil {
		log.Printf("Failed to clear all data for user %d: %v", userID, err)
		respondInternalError(w, r, "Failed to clear all data")
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *Server) exportUserData(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	payload, err := s.buildUserDataExport(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to export all data for user %d: %v", userID, err)
		respondInternalError(w, r, "Failed to export data")
		return
	}

	filename := fmt.Sprintf("symptomstracker-export-%d-%s.json", userID, time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		log.Printf("Failed to write export data for user %d: %v", userID, err)
	}
}

func defaultUserSettings() UserSettings {
	return UserSettings{
		Language:   "en",
		Theme:      "system",
		ScreenLock: "none",
		ShareTimer: "300",
	}
}

func (s *Server) loadUserSettings(ctx context.Context, userID int64) (UserSettings, error) {
	settings := defaultUserSettings()

	rows, err := s.queries.ListSettings(ctx, userID)
	if err != nil {
		return settings, err
	}

	for _, row := range rows {
		if !row.SettingsValue.Valid {
			continue
		}

		switch row.SettingsKey {
		case "language":
			settings.Language = row.SettingsValue.String
		case "theme":
			settings.Theme = row.SettingsValue.String
		case "screen_lock":
			settings.ScreenLock = row.SettingsValue.String
		case "share_timer":
			settings.ShareTimer = row.SettingsValue.String
		}
	}

	return settings, nil
}

func (s *Server) saveUserSettings(ctx context.Context, userID int64, settings UserSettings, createdAtUTC int64) error {
	upsert := func(key, value string) error {
		return s.queries.UpsertSetting(ctx, db.UpsertSettingParams{
			UserID:      userID,
			SettingsKey: key,
			SettingsValue: sql.NullString{
				String: value,
				Valid:  value != "",
			},
			CreatedAtUtc: createdAtUTC,
		})
	}

	if err := upsert("language", settings.Language); err != nil {
		return err
	}
	if err := upsert("theme", settings.Theme); err != nil {
		return err
	}
	if err := upsert("screen_lock", settings.ScreenLock); err != nil {
		return err
	}

	return upsert("share_timer", settings.ShareTimer)
}

func (s *Server) deleteAllUserData(ctx context.Context, userID int64) error {
	if err := s.queries.DeleteTrackableValuesByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.queries.DeleteTrackableDailyDismissalsByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.queries.DeleteEntriesByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.queries.DeleteTrackableDefinitionsByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.queries.DeleteSettingsByUser(ctx, userID); err != nil {
		return err
	}

	return s.queries.DeleteWebauthnCredentialsByUser(ctx, userID)
}

func (s *Server) buildUserDataExport(ctx context.Context, userID int64) (userDataExport, error) {
	settingsRows, err := s.queries.ListSettings(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	entries, err := s.queries.ListEntriesByUser(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	trackableValues, err := s.queries.ListTrackableValuesByUser(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	trackableDefinitions, err := s.queries.ListTrackableDefinitions(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	dismissals, err := s.queries.ListTrackableDailyDismissalsByUser(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	credentials, err := s.queries.ListWebauthnCredentialsByUser(ctx, userID)
	if err != nil {
		return userDataExport{}, err
	}

	settingsMap := make(map[string]*string, len(settingsRows))
	for _, setting := range settingsRows {
		settingsMap[setting.SettingsKey] = nullableString(setting.SettingsValue)
	}

	entryOut := make([]exportEntry, 0, len(entries))
	for _, entry := range entries {
		entryOut = append(entryOut, exportEntry{
			ID:                    entry.ID,
			RecordedAtUTC:         entry.RecordedAtUtc,
			TimezoneOffsetMinutes: entry.TimezoneOffsetMinutes,
			EntryDate:             entry.EntryDate,
			NoteText:              nullableString(entry.NoteText),
			IsPrivate:             entry.IsPrivate,
			CreatedAtUTC:          entry.CreatedAtUtc,
		})
	}

	valueOut := make([]exportTrackableValue, 0, len(trackableValues))
	for _, value := range trackableValues {
		valueOut = append(valueOut, exportTrackableValue{
			ID:                    value.ID,
			EntryID:               value.EntryID,
			TrackableDefinitionID: value.TrackableDefinitionID,
			ValueInt:              nullableInt64(value.ValueInt),
			ValueBool:             nullableInt64(value.ValueBool),
			ValueText:             nullableString(value.ValueText),
			LocationText:          nullableString(value.LocationText),
			NoteText:              nullableString(value.NoteText),
			EntryDate:             nullableString(value.EntryDate),
			CreatedAtUTC:          value.CreatedAtUtc,
			UpdatedAtUTC:          nullableInt64(value.UpdatedAtUtc),
		})
	}

	definitionOut := make([]exportTrackableDefinition, 0, len(trackableDefinitions))
	for _, definition := range trackableDefinitions {
		definitionOut = append(definitionOut, exportTrackableDefinition{
			ID:           definition.ID,
			TemplateID:   nullableInt64(definition.TemplateID),
			Name:         definition.Name,
			Icon:         nullableString(definition.Icon),
			ValueType:    definition.ValueType,
			Unit:         nullableString(definition.Unit),
			MinValue:     nullableInt64(definition.MinValue),
			MaxValue:     nullableInt64(definition.MaxValue),
			IsSensitive:  definition.IsSensitive,
			PrivateLabel: nullableString(definition.PrivateLabel),
			Category:     definition.Category,
			Active:       definition.Active,
			CreatedAtUTC: definition.CreatedAtUtc,
		})
	}

	credentialOut := make([]exportWebauthnCredential, 0, len(credentials))
	for _, credential := range credentials {
		credentialOut = append(credentialOut, exportWebauthnCredential{
			ID:           credential.ID,
			CredentialID: base64.StdEncoding.EncodeToString(credential.CredentialID),
			PublicKey:    base64.StdEncoding.EncodeToString(credential.PublicKey),
			SignCount:    credential.SignCount,
			Transports:   nullableString(credential.Transports),
			CreatedAtUTC: credential.CreatedAtUtc,
		})
	}

	return userDataExport{
		ExportedAtUTC:            time.Now().UTC().Unix(),
		UserID:                   userID,
		Entries:                  entryOut,
		TrackableValues:          valueOut,
		TrackableDefinitions:     definitionOut,
		TrackableDailyDismissals: dismissals,
		Settings:                 settingsMap,
		WebauthnCredentials:      credentialOut,
	}, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	v := value.String
	return &v
}

func nullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	v := value.Int64
	return &v
}
