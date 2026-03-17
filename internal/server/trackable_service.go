package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"roysland.me/symptomstracker/internal/db"
)

var errTrackableNotFound = errors.New("trackable not found")

type trackableSaveInput struct {
	TrackableID int64
	EntryID     int64
	HasEntryID  bool
	ValueInt    sql.NullInt64
	ValueBool   sql.NullInt64
	ValueText   sql.NullString
}

type trackableSaveResult struct {
	EntryID   int64
	ValueID   int64
	Timestamp int64
}

func (s *Server) buildTrackablePickerData(ctx context.Context, userID, entryID int64, hasEntryID bool, pickerID string, showAddTrackableLink bool) (trackablePickerViewData, error) {
	today := time.Now().Format("2006-01-02")
	trackableDefinitions, err := s.queries.ListTrackableDefinitionsWithDismissal(ctx, db.ListTrackableDefinitionsWithDismissalParams{
		DismissalDate: today,
		UserID:        userID,
	})
	if err != nil {
		return trackablePickerViewData{}, err
	}

	activeTrackables := make([]db.ListTrackableDefinitionsWithDismissalRow, 0, len(trackableDefinitions))
	dismissedTrackables := make([]db.ListTrackableDefinitionsWithDismissalRow, 0)
	for _, trackable := range trackableDefinitions {
		if trackable.DismissedToday == 1 {
			dismissedTrackables = append(dismissedTrackables, trackable)
			continue
		}
		activeTrackables = append(activeTrackables, trackable)
	}

	return trackablePickerViewData{
		EntryID:              entryID,
		HasEntryID:           hasEntryID,
		PickerID:             pickerID,
		ShowAddTrackableLink: showAddTrackableLink,
		ActiveTrackables:     activeTrackables,
		DismissedTrackables:  dismissedTrackables,
	}, nil
}

func (s *Server) saveTrackableValueForUser(ctx context.Context, userID int64, input trackableSaveInput, now time.Time) (trackableSaveResult, error) {
	trackable, err := s.queries.GetTrackableById(ctx, db.GetTrackableByIdParams{
		ID:     input.TrackableID,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return trackableSaveResult{}, errTrackableNotFound
	}
	if err != nil {
		return trackableSaveResult{}, err
	}

	nowUnix := now.UTC().Unix()
	result := trackableSaveResult{
		EntryID:   input.EntryID,
		Timestamp: nowUnix,
	}

	if input.HasEntryID {
		_, err = s.queries.GetEntryByID(ctx, db.GetEntryByIDParams{
			ID:     input.EntryID,
			UserID: userID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return trackableSaveResult{}, errEntryNotFound
		}
		if err != nil {
			return trackableSaveResult{}, err
		}

		existingValue, findErr := s.queries.FindEntryTrackableValue(ctx, db.FindEntryTrackableValueParams{
			EntryID:               input.EntryID,
			TrackableDefinitionID: input.TrackableID,
			UserID:                userID,
		})
		switch findErr {
		case nil:
			valueID, err := s.updateTrackableValueByType(ctx, trackable.ValueType, existingValue.ID, input, nowUnix)
			if err != nil {
				return trackableSaveResult{}, err
			}
			result.ValueID = valueID
		case sql.ErrNoRows:
			tv, err := s.createValueForEntry(ctx, input.EntryID, input.TrackableID, now, input.ValueInt, input.ValueBool, input.ValueText)
			if err != nil {
				return trackableSaveResult{}, err
			}
			result.ValueID = tv.ID
		default:
			return trackableSaveResult{}, findErr
		}

		return result, nil
	}

	if trackable.ValueType != "boolean" {
		cutoff := now.Add(-recentTrackableValueWindow).UTC().Unix()
		recent, findErr := s.queries.FindRecentTrackableValue(ctx, db.FindRecentTrackableValueParams{
			TrackableDefinitionID: input.TrackableID,
			UserID:                userID,
			CreatedAtUtc:          cutoff,
		})
		switch findErr {
		case nil:
			valueID, err := s.updateTrackableValueByType(ctx, trackable.ValueType, recent.ID, input, nowUnix)
			if err != nil {
				return trackableSaveResult{}, err
			}
			result.ValueID = valueID
			result.EntryID = recent.EntryID
			return result, nil
		case sql.ErrNoRows:
			tv, err := s.createEntryAndValue(ctx, userID, now, input.TrackableID, input.ValueInt, input.ValueBool, input.ValueText)
			if err != nil {
				return trackableSaveResult{}, err
			}
			result.ValueID = tv.ID
			result.EntryID = tv.EntryID
			return result, nil
		default:
			return trackableSaveResult{}, findErr
		}
	}

	tv, err := s.createEntryAndValue(ctx, userID, now, input.TrackableID, input.ValueInt, input.ValueBool, input.ValueText)
	if err != nil {
		return trackableSaveResult{}, err
	}
	result.ValueID = tv.ID
	result.EntryID = tv.EntryID
	return result, nil
}

func (s *Server) updateTrackableValueByType(ctx context.Context, valueType string, valueID int64, input trackableSaveInput, nowUnix int64) (int64, error) {
	switch valueType {
	case "integer":
		updated, err := s.queries.UpdateTrackableValueInt(ctx, db.UpdateTrackableValueIntParams{
			ValueInt:     input.ValueInt,
			UpdatedAtUtc: sql.NullInt64{Int64: nowUnix, Valid: true},
			ID:           valueID,
		})
		if err != nil {
			return 0, err
		}
		return updated.ID, nil
	case "text":
		updated, err := s.queries.UpdateTrackableValueText(ctx, db.UpdateTrackableValueTextParams{
			ValueText:    input.ValueText,
			UpdatedAtUtc: sql.NullInt64{Int64: nowUnix, Valid: true},
			ID:           valueID,
		})
		if err != nil {
			return 0, err
		}
		return updated.ID, nil
	default:
		return valueID, nil
	}
}

func (s *Server) saveTrackableDismissalForUser(ctx context.Context, userID, trackableID int64, dismissed bool, now time.Time) (db.TrackableDailyDismissal, error) {
	_, err := s.queries.GetTrackableById(ctx, db.GetTrackableByIdParams{
		ID:     trackableID,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.TrackableDailyDismissal{}, errTrackableNotFound
	}
	if err != nil {
		return db.TrackableDailyDismissal{}, err
	}

	return s.queries.UpsertTrackableDailyDismissal(ctx, db.UpsertTrackableDailyDismissalParams{
		UserID:                userID,
		TrackableDefinitionID: trackableID,
		DismissalDate:         now.Format("2006-01-02"),
		Dismissed:             boolToInt64(dismissed),
		CreatedAtUtc:          now.UTC().Unix(),
		UpdatedAtUtc:          now.UTC().Unix(),
	})
}

func (s *Server) createEntryAndValue(
	ctx context.Context,
	userID int64,
	now time.Time,
	trackableID int64,
	valueInt sql.NullInt64,
	valueBool sql.NullInt64,
	valueText sql.NullString,
) (db.TrackableValue, error) {
	entry, err := s.createEntry(ctx, userID, now, sql.NullString{}, 0)
	if err != nil {
		return db.TrackableValue{}, fmt.Errorf("create entry: %w", err)
	}

	log.Printf("Created entry %d for trackable %d", entry.ID, trackableID)

	return s.createValueForEntry(ctx, entry.ID, trackableID, now, valueInt, valueBool, valueText)
}

func (s *Server) createValueForEntry(
	ctx context.Context,
	entryID int64,
	trackableID int64,
	now time.Time,
	valueInt sql.NullInt64,
	valueBool sql.NullInt64,
	valueText sql.NullString,
) (db.TrackableValue, error) {
	tv, err := s.queries.CreateTrackableValue(ctx, db.CreateTrackableValueParams{
		EntryID:               entryID,
		TrackableDefinitionID: trackableID,
		ValueInt:              valueInt,
		ValueBool:             valueBool,
		ValueText:             valueText,
		CreatedAtUtc:          now.UTC().Unix(),
	})
	if err != nil {
		return db.TrackableValue{}, fmt.Errorf("create trackable value: %w", err)
	}
	return tv, nil
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
