package i18n

import "fmt"

// Default is the language used when no preference is detected or the
// requested language isn't supported.
const Default = "en"

var dictionaries = map[string]map[string]string{
	"en": en,
	"tr": tr,
}

// Supported reports whether lang has a dictionary.
func Supported(lang string) bool {
	_, ok := dictionaries[lang]
	return ok
}

// Translator returns a lookup function bound to lang, for use in templates
// via {{call .T "some.key"}} or {{call .T "some.key" arg1 arg2}}. Keys
// missing from lang fall back to the default dictionary, then to the raw key
// itself (so a missing translation is visible rather than silently blank).
func Translator(lang string) func(string, ...any) string {
	if !Supported(lang) {
		lang = Default
	}
	dict := dictionaries[lang]

	return func(key string, args ...any) string {
		format, ok := dict[key]
		if !ok {
			format, ok = dictionaries[Default][key]
			if !ok {
				return key
			}
		}
		if len(args) == 0 {
			return format
		}
		return fmt.Sprintf(format, args...)
	}
}

// T is a convenience one-shot translation, for Go-side messages (handler
// flash/validation text) where building a Translator closure would be overkill.
func T(lang, key string, args ...any) string {
	return Translator(lang)(key, args...)
}
