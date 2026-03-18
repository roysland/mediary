package server

import (
	"database/sql"

	"roysland.me/symptomstracker/internal/db"
)

type entryTrackableValueView struct {
	Name  string
	Icon  string
	Value string
}

type entryView struct {
	ID                  int64
	RecordedAtUtc       int64
	EntryDate           string
	NoteText            sql.NullString
	IsPrivate           int64
	IsDraft             bool
	AudioFilePath       sql.NullString
	TranscriptionStatus string
	Trackables          []entryTrackableValueView
}

type entryWithTrackableRow struct {
	ID                  int64
	RecordedAtUtc       int64
	EntryDate           string
	NoteText            sql.NullString
	IsPrivate           int64
	IsDraft             int64
	AudioFilePath       sql.NullString
	TranscriptionStatus string
	TrackableValueID    sql.NullInt64
	TrackableName       sql.NullString
	TrackableIcon       sql.NullString
	TrackableValueType  sql.NullString
	ValueInt            sql.NullInt64
	ValueBool           sql.NullInt64
	ValueText           sql.NullString
	TrackableUnit       sql.NullString
}

func buildEntryViews(rows []db.ListEntriesRow) []entryView {
	viewRows := make([]entryWithTrackableRow, 0, len(rows))
	for _, row := range rows {
		viewRows = append(viewRows, entryWithTrackableRow{
			ID:                  row.ID,
			RecordedAtUtc:       row.RecordedAtUtc,
			EntryDate:           row.EntryDate,
			NoteText:            row.NoteText,
			IsPrivate:           row.IsPrivate,
			IsDraft:             row.IsDraft,
			AudioFilePath:       row.AudioFilePath,
			TranscriptionStatus: row.TranscriptionStatus,
			TrackableValueID:    row.TrackableValueID,
			TrackableName:       row.TrackableName,
			TrackableIcon:       row.TrackableIcon,
			TrackableValueType:  row.TrackableValueType,
			ValueInt:            row.ValueInt,
			ValueBool:           row.ValueBool,
			ValueText:           row.ValueText,
			TrackableUnit:       row.TrackableUnit,
		})
	}

	return buildEntryViewsFromRows(viewRows)
}

func buildEntryView(rows []db.GetEntryWithTrackablesRow) (entryView, bool) {
	viewRows := make([]entryWithTrackableRow, 0, len(rows))
	for _, row := range rows {
		viewRows = append(viewRows, entryWithTrackableRow{
			ID:                  row.ID,
			RecordedAtUtc:       row.RecordedAtUtc,
			EntryDate:           row.EntryDate,
			NoteText:            row.NoteText,
			IsPrivate:           row.IsPrivate,
			IsDraft:             row.IsDraft,
			AudioFilePath:       row.AudioFilePath,
			TranscriptionStatus: row.TranscriptionStatus,
			TrackableValueID:    row.TrackableValueID,
			TrackableName:       row.TrackableName,
			TrackableIcon:       row.TrackableIcon,
			TrackableValueType:  row.TrackableValueType,
			ValueInt:            row.ValueInt,
			ValueBool:           row.ValueBool,
			ValueText:           row.ValueText,
			TrackableUnit:       row.TrackableUnit,
		})
	}

	entries := buildEntryViewsFromRows(viewRows)
	if len(entries) == 0 {
		return entryView{}, false
	}

	return entries[0], true
}

func buildEntryViewsFromRows(rows []entryWithTrackableRow) []entryView {
	entries := make([]entryView, 0)
	entryIndex := make(map[int64]int)
	for _, row := range rows {
		index, exists := entryIndex[row.ID]
		if !exists {
			entries = append(entries, entryView{
				ID:                  row.ID,
				RecordedAtUtc:       row.RecordedAtUtc,
				EntryDate:           row.EntryDate,
				NoteText:            row.NoteText,
				IsPrivate:           row.IsPrivate,
				IsDraft:             row.IsDraft == 1,
				AudioFilePath:       row.AudioFilePath,
				TranscriptionStatus: row.TranscriptionStatus,
			})
			index = len(entries) - 1
			entryIndex[row.ID] = index
		}

		if !row.TrackableValueID.Valid {
			continue
		}

		entries[index].Trackables = append(entries[index].Trackables, entryTrackableValueView{
			Name: row.TrackableName.String,
			Icon: row.TrackableIcon.String,
			Value: formatTrackableValue(trackableValueFields{
				valueType: row.TrackableValueType.String,
				valueInt:  row.ValueInt,
				valueBool: row.ValueBool,
				valueText: row.ValueText,
				unit:      row.TrackableUnit,
			}),
		})
	}

	return entries
}
