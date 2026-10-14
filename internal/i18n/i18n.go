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
//
// When the last argument is the int 1 and a "<key>.one" entry exists, that
// singular form is used instead ("1 day" rather than "1 days"). Languages
// without a singular form (Turkish) simply don't define ".one" keys.
func Translator(lang string) func(string, ...any) string {
	if !Supported(lang) {
		lang = Default
	}
	dict := dictionaries[lang]

	lookup := func(key string) (string, bool) {
		if format, ok := dict[key]; ok {
			return format, true
		}
		format, ok := dictionaries[Default][key]
		return format, ok
	}

	return func(key string, args ...any) string {
		if n, ok := lastInt(args); ok && n == 1 {
			if format, ok := dict[key+".one"]; ok {
				return fmt.Sprintf(format, args...)
			}
		}
		format, ok := lookup(key)
		if !ok {
			return key
		}
		if len(args) == 0 {
			return format
		}
		return fmt.Sprintf(format, args...)
	}
}

func lastInt(args []any) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	n, ok := args[len(args)-1].(int)
	return n, ok
}

// T is a convenience one-shot translation, for Go-side messages (handler
// flash/validation text) where building a Translator closure would be overkill.
func T(lang, key string, args ...any) string {
	return Translator(lang)(key, args...)
}
