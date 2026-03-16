package i18n

func T(key string) string {

	switch key {

	case "app.title":
		return "Symptoms Tracker"

	case "entry.input.label":
		return "How are you feeling?"

	case "entry.add_trackables":
		return "Add Trackables"

	case "entry.add_trackable_link":
		return "Add Trackable"

	case "entry.view_trackables":
		return "View tracked items"

	case "entry.private.label":
		return "Private/Sensitive."

	case "entry.add_trackable":
		return "Add trackable"

	case "entry.filter.placeholder":
		return "Filter symptoms and activities"

	case "save":
		return "Save"

	case "nav.home":
		return "Home"

	case "nav.entries":
		return "Entries"

	case "nav.settings":
		return "Settings"

	case "nav.trackables":
		return "Trackables"

	case "settings.language.label":
		return "Language"

	case "settings.language.english":
		return "English"

	case "settings.language.norwegian":
		return "Norwegian"

	case "settings.clear_data":
		return "Clear all data"

	case "settings.danger_zone":
		return "Danger zone"

	case "settings.clear_data_confirm":
		return "Are you sure you want to clear all data? This action cannot be undone."

	case "settings.theme.label":
		return "Theme"

	case "settings.theme.light":
		return "Light"

	case "settings.theme.dark":
		return "Dark"

	case "settings.theme.system":
		return "System default"

	case "settings.screen_lock.label":
		return "Screen lock"

	case "settings.screen_lock.never":
		return "Never"

	case "settings.screen_lock.1_minute":
		return "After 1 minute"

	case "settings.screen_lock.5_minutes":
		return "After 5 minutes"

	case "settings.screen_lock.10_minutes":
		return "After 10 minutes"

	case "settings.share_timer.label":
		return "Share timer"

	case "settings.share_timer.never":
		return "Never"

	case "settings.share_timer.5_minutes":
		return "5 minutes"

	case "settings.share_timer.10_minutes":
		return "10 minutes"

	case "settings.share_timer.30_minutes":
		return "30 minutes"

	case "trackable.name.label":
		return "Name of symptom or activity"

	case "trackable.private.label":
		return "What will the private label be?"

	case "trackable.value_type.label":
		return "Value type"

	case "trackable.value_type.integer":
		return "Integer"

	case "trackable.value_type.boolean":
		return "Boolean"

	case "trackable.value_type.text":
		return "Text"

	case "trackable.sensitive.label":
		return "Sensitive"

	case "trackable.advanced_options":
		return "Advanced options"

	case "trackable.icon.label":
		return "Icon"

	case "trackable.icon.placeholder":
		return "Emoji or short text (e.g. '💧' or 'Water')"

	case "trackable.min_value.label":
		return "Minimum value"

	case "trackable.min_value.placeholder":
		return "Only for numeric trackables. Optional."

	case "trackable.max_value.label":
		return "Maximum value"

	case "trackable.max_value.placeholder":
		return "Only for numeric trackables. Optional."

	case "trackable.unit.label":
		return "Unit"

	case "trackable.unit.placeholder":
		return "e.g. 'liters', 'hours', 'mood'"

	case "trackable.category.label":
		return "Category"

	case "trackable.category.placeholder":
		return "e.g. 'Health', 'Mood', 'Activity'"

	case "trackable.no_trackable_tracked":
		return "No symptoms or activities tracked yet."

	case "trackable.color":
		return "Color"

	case "trackable.category.default":
		return "Default"

	case "trackable.category.symptom":
		return "Symptom"

	case "trackable.category.activity":
		return "Activity"

	case "trackable.category.measurement":
		return "Measurement"

	case "trackable.category.state":
		return "State"

	default:
		return key
	}

}
