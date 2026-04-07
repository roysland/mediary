package server

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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
	d, err := time.Parse(dateLayoutISO, dayStr)
	if err != nil {
		return time.Time{}, err
	}
	if d.Format(dateLayoutISO) > now.Format(dateLayoutISO) {
		return now, nil
	}
	return d, nil
}

func buildDayNavigation(selectedDay time.Time, now time.Time) []dayNav {
	todayStr := now.Format(dateLayoutISO)
	selectedDayStr := selectedDay.Format(dateLayoutISO)
	navigation := make([]dayNav, 0, dayNavigationPastDays+dayNavigationFutureDays+1)
	for offset := -dayNavigationPastDays; offset <= dayNavigationFutureDays; offset++ {
		d := selectedDay.AddDate(0, 0, offset)
		dateStr := d.Format(dateLayoutISO)
		if dateStr > todayStr {
			break
		}
		navigation = append(navigation, dayNav{
			Date:      dateStr,
			DayNumber: d.Format("02"),
			DayName:   d.Format("Mon"),
			IsCurrent: dateStr == selectedDayStr,
		})
	}
	return navigation
}

func resolveEntryDate(value string, now time.Time) (string, error) {
	selectedDay, err := parseSelectedDay(strings.TrimSpace(value), now)
	if err != nil {
		return "", err
	}
	return selectedDay.Format(dateLayoutISO), nil
}

func (s *Server) createEntry(
	ctx context.Context,
	userID int64,
	now time.Time,
	entryDate string,
	note sql.NullString,
	isPrivate int64,
) (db.Entry, error) {
	return s.queries.CreateEntry(ctx, db.CreateEntryParams{
		UserID:                userID,
		RecordedAtUtc:         now.UTC().Unix(),
		TimezoneOffsetMinutes: defaultTimezoneOffsetMinutes,
		EntryDate:             entryDate,
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

	entries := buildEntryViews(rows)
	if err := s.attachImagesToEntryViews(ctx, userID, entries); err != nil {
		return nil, err
	}

	return entries, nil
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

	images, err := s.queries.GetImagesByEntryID(ctx, db.GetImagesByEntryIDParams{
		EntryID: entry.ID,
		UserID:  userID,
	})
	if err != nil {
		return entryView{}, err
	}

	entry.Images = mapEntryImages(images)

	return entry, nil
}

func (s *Server) attachImagesToEntryViews(ctx context.Context, userID int64, entries []entryView) error {
	for i := range entries {
		images, err := s.queries.GetImagesByEntryID(ctx, db.GetImagesByEntryIDParams{
			EntryID: entries[i].ID,
			UserID:  userID,
		})
		if err != nil {
			return err
		}

		entries[i].Images = mapEntryImages(images)
	}

	return nil
}

func mapEntryImages(images []db.EntryImage) []entryImageView {
	out := make([]entryImageView, 0, len(images))
	for _, img := range images {
		out = append(out, entryImageView{
			ID:       img.ID,
			FilePath: img.FilePath,
			MimeType: img.MimeType,
		})
	}
	return out
}
