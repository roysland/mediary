package server

import "time"

const (
	defaultTimezoneOffsetMinutes = 0

	dateLayoutISO     = "2006-01-02"
	dateTimeLayoutUTC = "2006-01-02 15:04:05"

	dayNavigationPastDays   = 4
	dayNavigationFutureDays = 1

	recentTrackableValueWindow = 30 * time.Second

	// Keep a small delay for development UX simulation; disabled in production.
	devAddEntryDelay = 500 * time.Millisecond
)
