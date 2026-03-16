package server

import (
	"context"
	"database/sql"
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

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		settings, err := s.loadUserSettings(r.Context(), userID)
		if err != nil {
			respondInternalError(w, r, "Failed to load settings")
			return
		}

		s.renderPage(w, r, "settings_title", "settings_content", map[string]any{
			"Settings": settings,
		})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			respondBadRequest(w, r, "Invalid form data")
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
	default:
		respondMethodNotAllowed(w, r)
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
