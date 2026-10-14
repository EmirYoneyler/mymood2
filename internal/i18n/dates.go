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

// weekdayAbbr is Monday-first, matching the calendar grid layout.
var weekdayAbbr = map[string][]string{
	"en": {"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	"tr": {"Pzt", "Sal", "Çar", "Per", "Cum", "Cmt", "Paz"},
}

// weekdayFull is indexed by time.Weekday (Sunday = 0).
var weekdayFull = map[string][]string{
	"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
	"tr": {"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"},
}

func resolve(lang string) string {
	if !Supported(lang) {
		return Default
	}
	return lang
}

// MonthAbbrs returns the 12 short month names (Jan..Dec / Oca..Ara) for lang.
func MonthAbbrs(lang string) []string {
	return monthAbbr[resolve(lang)]
}

// MonthNames returns the 12 full month names for lang.
func MonthNames(lang string) []string {
	return monthFull[resolve(lang)]
}

// WeekdayAbbrs returns the 7 short weekday names, Monday first.
func WeekdayAbbrs(lang string) []string {
	return weekdayAbbr[resolve(lang)]
}

// FormatDate renders a date as "5 Jun 2026" (en) or "5 Haz 2026" (tr).
func FormatDate(lang string, t time.Time) string {
	abbr := MonthAbbrs(lang)
	return fmt.Sprintf("%d %s %d", t.Day(), abbr[int(t.Month())-1], t.Year())
}

// FormatDayMonth renders a date without the year: "5 June" / "5 Haziran".
func FormatDayMonth(lang string, t time.Time) string {
	return fmt.Sprintf("%d %s", t.Day(), MonthNames(lang)[int(t.Month())-1])
}

// FormatLongDate renders a date with its weekday, e.g. "Wednesday, 14 October"
// (en) or "14 Ekim, Çarşamba" (tr). The year is appended when withYear is set.
func FormatLongDate(lang string, t time.Time, withYear bool) string {
	lang = resolve(lang)
	day := FormatDayMonth(lang, t)
	if withYear {
		day = fmt.Sprintf("%s %d", day, t.Year())
	}
	weekday := weekdayFull[lang][int(t.Weekday())]
	if lang == "tr" {
		return day + ", " + weekday
	}
	return weekday + ", " + day
}

// FormatMonthYear renders "October 2026" / "Ekim 2026".
func FormatMonthYear(lang string, t time.Time) string {
	return fmt.Sprintf("%s %d", MonthNames(lang)[int(t.Month())-1], t.Year())
}
