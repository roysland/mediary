package server

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

var errEntryNotFound = errors.New("entry not found")

type dayNav struct {
	Date      string
	DayNumber string
	DayName   string
	IsCurrent bool
}

func parseSelectedDay(dayStr string, now time.Time) (time.Time, error) {
	if dayStr == "" {
		return now, nil
	}
	return time.Parse("2006-01-02", dayStr)
}

func buildDayNavigation(selectedDay time.Time) []dayNav {
	selectedDayStr := selectedDay.Format("2006-01-02")
	navigation := make([]dayNav, 0, dayNavigationPastDays+dayNavigationFutureDays+1)
	for offset := -dayNavigationPastDays; offset <= dayNavigationFutureDays; offset++ {
		d := selectedDay.AddDate(0, 0, offset)
		dateStr := d.Format("2006-01-02")
		navigation = append(navigation, dayNav{
			Date:      dateStr,
			DayNumber: d.Format("02"),
			DayName:   d.Format("Mon"),
			IsCurrent: dateStr == selectedDayStr,
		})
	}
	return navigation
}

func (s *Server) createEntry(
	ctx context.Context,
	userID int64,
	now time.Time,
	note sql.NullString,
	isPrivate int64,
) (db.Entry, error) {
	return s.queries.CreateEntry(ctx, db.CreateEntryParams{
		UserID:                userID,
		RecordedAtUtc:         now.UTC().Unix(),
		TimezoneOffsetMinutes: defaultTimezoneOffsetMinutes,
		EntryDate:             now.Format("2006-01-02"),
		NoteText:              note,
		IsPrivate:             isPrivate,
		CreatedAtUtc:          now.UTC().Unix(),
	})
}

func (s *Server) listEntryViewsByDay(ctx context.Context, userID int64, day string) ([]entryView, error) {
	rows, err := s.queries.ListEntries(ctx, db.ListEntriesParams{
		UserID:    userID,
		EntryDate: day,
	})
	if err != nil {
		return nil, err
	}
	return buildEntryViews(rows), nil
}

func (s *Server) loadEntryViewByID(ctx context.Context, userID, entryID int64) (entryView, error) {
	rows, err := s.queries.GetEntryWithTrackables(ctx, db.GetEntryWithTrackablesParams{
		UserID: userID,
		ID:     entryID,
	})
	if err != nil {
		return entryView{}, err
	}

	if len(rows) == 0 {
		_, err := s.queries.GetEntryByID(ctx, db.GetEntryByIDParams{
			ID:     entryID,
			UserID: userID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return entryView{}, errEntryNotFound
		}
		if err != nil {
			return entryView{}, err
		}
		return entryView{}, errEntryNotFound
	}

	entry, ok := buildEntryView(rows)
	if !ok {
		return entryView{}, errEntryNotFound
	}

	return entry, nil
}
