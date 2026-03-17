package server

import "time"

const (
	defaultTimezoneOffsetMinutes = 0

	dayNavigationPastDays   = 4
	dayNavigationFutureDays = 1

	recentTrackableValueWindow = 30 * time.Second

	// Keep a small delay for development UX simulation; disabled in production.
	devAddEntryDelay = 500 * time.Millisecond
)
