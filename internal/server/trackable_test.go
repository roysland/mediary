package server

import (
	"database/sql"
	"testing"
)

func TestBoolToInt64(t *testing.T) {
	if got := boolToInt64(true); got != 1 {
		t.Fatalf("expected 1 for true, got %d", got)
	}
	if got := boolToInt64(false); got != 0 {
		t.Fatalf("expected 0 for false, got %d", got)
	}
}

func TestFormatTrackableValue(t *testing.T) {
	tests := []struct {
		name   string
		fields trackableValueFields
		want   string
	}{
		{
			name: "integer with unit",
			fields: trackableValueFields{
				valueType: "integer",
				valueInt:  sql.NullInt64{Int64: 42, Valid: true},
				unit:      sql.NullString{String: "bpm", Valid: true},
			},
			want: "42 bpm",
		},
		{
			name: "integer without unit",
			fields: trackableValueFields{
				valueType: "integer",
				valueInt:  sql.NullInt64{Int64: 7, Valid: true},
			},
			want: "7",
		},
		{
			name: "boolean",
			fields: trackableValueFields{
				valueType: "boolean",
			},
			want: "Yes",
		},
		{
			name: "text",
			fields: trackableValueFields{
				valueType: "text",
				valueText: sql.NullString{String: "moderate", Valid: true},
			},
			want: "moderate",
		},
		{
			name: "fallback text first",
			fields: trackableValueFields{
				valueType: "unknown",
				valueText: sql.NullString{String: "fallback", Valid: true},
				valueInt:  sql.NullInt64{Int64: 9, Valid: true},
			},
			want: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTrackableValue(tt.fields)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
