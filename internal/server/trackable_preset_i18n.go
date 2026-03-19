package server

import (
	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

var trackablePresetNameToI18nKey = map[string]string{
	"Fatigue":                 "trackable.preset.fatigue",
	"Post-exertional malaise": "trackable.preset.post_exertional_malaise",
	"Brain fog":               "trackable.preset.brain_fog",
	"Sleep quality":           "trackable.preset.sleep_quality",
	"Pain":                    "trackable.preset.pain",
	"Heart rate":              "trackable.preset.heart_rate",
	"Blood pressure":          "trackable.preset.blood_pressure",
	"Palpitations":            "trackable.preset.palpitations",
	"Dizziness":               "trackable.preset.dizziness",
	"Physical activity":       "trackable.preset.physical_activity",
	"Steps":                   "trackable.preset.steps",
	"Hours slept":             "trackable.preset.hours_slept",
	"Unrefreshing sleep":      "trackable.preset.unrefreshing_sleep",
	"Headache":                "trackable.preset.headache",
	"Light sensitivity":       "trackable.preset.light_sensitivity",
	"Sound sensitivity":       "trackable.preset.sound_sensitivity",
	"Temperature":             "trackable.preset.temperature",
	"Feeling feverish":        "trackable.preset.feeling_feverish",
	"Nausea":                  "trackable.preset.nausea",
	"Mood":                    "trackable.preset.mood",
	"Anxiety":                 "trackable.preset.anxiety",
	"Sexual activity":         "trackable.preset.sexual_activity",
}

func localizeTrackablePresetName(locale, presetName string) string {
	key, ok := trackablePresetNameToI18nKey[presetName]
	if !ok {
		return presetName
	}

	translated := i18n.TForLocale(locale, key)
	if translated == key {
		return presetName
	}

	return translated
}

func localizeTrackableTemplates(locale string, templates []db.TrackableTemplate) []db.TrackableTemplate {
	localized := make([]db.TrackableTemplate, len(templates))
	copy(localized, templates)

	for i := range localized {
		localized[i].Name = localizeTrackablePresetName(locale, localized[i].Name)
	}

	return localized
}
