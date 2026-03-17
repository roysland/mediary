package i18n

import "sort"

const (
	LocaleEnglish   = "en"
	LocaleNorwegian = "no"
	DefaultLocale   = LocaleEnglish
)

type Catalog map[string]string

var catalogs = map[string]Catalog{
	LocaleEnglish: englishCatalog,
}

func T(key string) string {
	return TForLocale(DefaultLocale, key)
}

func TForLocale(locale, key string) string {
	if value, ok := Lookup(locale, key); ok {
		return value
	}

	if locale != DefaultLocale {
		if value, ok := Lookup(DefaultLocale, key); ok {
			return value
		}
	}

	return key
}

func Lookup(locale, key string) (string, bool) {
	catalog, ok := catalogs[locale]
	if !ok {
		return "", false
	}

	value, ok := catalog[key]
	return value, ok
}

func HasKey(locale, key string) bool {
	_, ok := Lookup(locale, key)
	return ok
}

func Locales() []string {
	locales := make([]string, 0, len(catalogs))
	for locale := range catalogs {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

func Keys(locale string) []string {
	catalog, ok := catalogs[locale]
	if !ok {
		return nil
	}

	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
