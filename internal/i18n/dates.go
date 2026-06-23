package i18n

import (
	"fmt"
	"time"
)

var monthAbbr = map[string][]string{
	"en": {"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	"tr": {"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"},
}

var monthFull = map[string][]string{
	"en": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	"tr": {"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"},
}

// MonthAbbrs returns the 12 short month names (Jan..Dec / Oca..Ara) for lang.
func MonthAbbrs(lang string) []string {
	if !Supported(lang) {
		lang = Default
	}
	return monthAbbr[lang]
}

// MonthNames returns the 12 full month names for lang.
func MonthNames(lang string) []string {
	if !Supported(lang) {
		lang = Default
	}
	return monthFull[lang]
}

// FormatDate renders a date as "15 Jun 2026" (en) or "15 Haz 2026" (tr).
func FormatDate(lang string, t time.Time) string {
	abbr := MonthAbbrs(lang)
	return fmt.Sprintf("%02d %s %d", t.Day(), abbr[int(t.Month())-1], t.Year())
}
