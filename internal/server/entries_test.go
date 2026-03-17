package server

import (
	"testing"
	"time"
)

func TestParseSelectedDay(t *testing.T) {
	now := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "empty uses now",
			input: "",
			want:  now,
		},
		{
			name:  "valid day",
			input: "2026-03-01",
			want:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid day",
			input:   "03-01-2026",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSelectedDay(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestBuildDayNavigation(t *testing.T) {
	selected := time.Date(2026, 3, 17, 9, 30, 0, 0, time.UTC)
	nav := buildDayNavigation(selected)

	wantLen := dayNavigationPastDays + dayNavigationFutureDays + 1
	if len(nav) != wantLen {
		t.Fatalf("expected %d nav items, got %d", wantLen, len(nav))
	}

	currentDate := selected.Format("2006-01-02")
	currentCount := 0
	for _, d := range nav {
		if d.IsCurrent {
			currentCount++
			if d.Date != currentDate {
				t.Fatalf("current item has wrong date: got %s want %s", d.Date, currentDate)
			}
		}
	}

	if currentCount != 1 {
		t.Fatalf("expected exactly one current item, got %d", currentCount)
	}

	if nav[0].Date != selected.AddDate(0, 0, -dayNavigationPastDays).Format("2006-01-02") {
		t.Fatalf("unexpected first date: %s", nav[0].Date)
	}
	if nav[len(nav)-1].Date != selected.AddDate(0, 0, dayNavigationFutureDays).Format("2006-01-02") {
		t.Fatalf("unexpected last date: %s", nav[len(nav)-1].Date)
	}
}
