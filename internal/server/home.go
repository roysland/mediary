package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

type homeAlert struct {
	Version string
	Key     string
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		respondNotFound(w, r, "Not found")
		return
	}

	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	alert, err := s.activeAlertForUser(r.Context(), userID)
	if err != nil {
		respondInternalError(w, r, "Failed to load home alert")
		return
	}

	today := time.Now().Format(dateLayoutISO)
	s.renderPage(w, r, "home_title", "home_content", map[string]interface{}{
		"SelectedDay": today,
		"TodayStr":    today,
		"Alert":       alert,
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

func (s *Server) dismissAlert(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	version := strings.TrimSpace(r.PathValue("version"))
	if version == "" {
		respondBadRequest(w, r, "Invalid alert version")
		return
	}

	err := s.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		UserID:      userID,
		SettingsKey: fmt.Sprintf("alert_dismissed_%s", version),
		SettingsValue: sql.NullString{
			String: "1",
			Valid:  true,
		},
		CreatedAtUtc: time.Now().UTC().Unix(),
	})
	if err != nil {
		respondInternalError(w, r, "Failed to dismiss alert")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) activeAlertForUser(ctx context.Context, userID int64) (*homeAlert, error) {
	if activeAlertVersion == "" {
		return nil, nil
	}

	setting, err := s.queries.GetSetting(ctx, db.GetSettingParams{
		UserID:      userID,
		SettingsKey: fmt.Sprintf("alert_dismissed_%s", activeAlertVersion),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return &homeAlert{Version: activeAlertVersion, Key: activeAlertI18nKey}, nil
		}
		return nil, err
	}

	if setting.SettingsValue.Valid && setting.SettingsValue.String == "1" {
		return nil, nil
	}

	return &homeAlert{Version: activeAlertVersion, Key: activeAlertI18nKey}, nil
}
