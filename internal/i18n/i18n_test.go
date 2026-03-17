package i18n

import "testing"

func TestTReturnsExpectedValuesForKnownKeys(t *testing.T) {
	tests := map[string]string{
		"app.title":                 "Symptoms Tracker",
		"entries.title":             "Entries",
		"settings.clear_data":       "Clear all data",
		"trackable.value_type.text": "Text",
	}

	for key, want := range tests {
		if got := T(key); got != want {
			t.Fatalf("key %q: expected %q, got %q", key, want, got)
		}
	}
}

func TestTFallsBackToKeyForUnknownValues(t *testing.T) {
	const key = "this.key.does.not.exist"
	if got := T(key); got != key {
		t.Fatalf("expected fallback to key, got %q", got)
	}
}
