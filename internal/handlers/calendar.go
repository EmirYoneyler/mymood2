package handlers

import (
	"fmt"
	"math"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/models"
)

const dateParamLayout = "2006-01-02"
const monthParamLayout = "2006-01"

// maxYearsBack bounds how far back you can browse/rate the calendar archive.
const maxYearsBack = 80

type rgbColor struct{ R, G, B int }

func (c rgbColor) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// scoreGradientStops maps a 1-10 score to a continuous color: red at the
// bottom, through orange and amber, to green, ending in teal at the top.
// Stops are evenly spaced so every tenth of a point produces a slightly
// different shade. The same stops are mirrored in app.css (.range track,
// legend) and app.js (live preview) - keep them in sync.
var scoreGradientStops = []struct {
	Pos   float64
	Color rgbColor
}{
	{1.00, rgbColor{229, 72, 77}},   // red
	{3.25, rgbColor{247, 107, 21}},  // orange
	{5.50, rgbColor{255, 197, 61}},  // amber
	{7.75, rgbColor{70, 167, 88}},   // green
	{10.00, rgbColor{18, 165, 148}}, // teal
}

func scoreGradient(score float64) rgbColor {
	stops := scoreGradientStops
	if score <= stops[0].Pos {
		return stops[0].Color
	}
	for i := 1; i < len(stops); i++ {
		if score <= stops[i].Pos {
			prev, next := stops[i-1], stops[i]
			t := (score - prev.Pos) / (next.Pos - prev.Pos)
			return rgbColor{
				R: lerp(prev.Color.R, next.Color.R, t),
				G: lerp(prev.Color.G, next.Color.G, t),
				B: lerp(prev.Color.B, next.Color.B, t),
			}
		}
	}
	return stops[len(stops)-1].Color
}

func lerp(a, b int, t float64) int {
	return a + int(math.Round(float64(b-a)*t))
}

// scoreLabel formats a score compactly for tight spaces like calendar cells:
// one decimal place, except a perfect 10 which is shown as "10".
func scoreLabel(score float64) string {
	if score >= 10 {
		return "10"
	}
	return fmt.Sprintf("%.1f", score)
}

// bucketFor classifies a score into one of five named buckets, used for the
// word shown next to the score picker ("Great", "Low", ...).
func bucketFor(score float64) string {
	switch {
	case score >= 9:
		return "excellent"
	case score >= 7:
		return "great"
	case score >= 5:
		return "good"
	case score >= 3:
		return "low"
	default:
		return "poor"
	}
}

// avatarInitial returns the uppercased first letter of a username.
func avatarInitial(username string) string {
	r, _ := utf8.DecodeRuneInString(username)
	if r == utf8.RuneError {
		return "?"
	}
	return string(unicode.ToUpper(r))
}

// avatarTone picks one of a handful of muted avatar colors, stable per username.
func avatarTone(username string) int {
	var h uint32 = 2166136261
	for i := 0; i < len(username); i++ {
		h ^= uint32(username[i])
		h *= 16777619
	}
	return int(h % 6)
}

// TemplateFuncs are helpers available in every template.
func TemplateFuncs() map[string]interface{} {
	return map[string]interface{}{
		"scoreColor": func(score float64) string { return scoreGradient(score).Hex() },
		"scoreLabel": scoreLabel,
		"initial":    avatarInitial,
		"avatarTone": avatarTone,
	}
}

func startOfWeek(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7 // days since Monday
	return t.AddDate(0, 0, -offset)
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
}

func clampYear(year, min, max int) int {
	if year < min {
		return min
	}
	if year > max {
		return max
	}
	return year
}

func summarizeEntries(entries []models.MoodEntry) (average float64, count int) {
	count = len(entries)
	if count == 0 {
		return 0, 0
	}
	var sum float64
	for _, e := range entries {
		sum += e.Score
	}
	return sum / float64(count), count
}

func formatAverage(average float64, count int) string {
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f", average)
}

// --- Month calendar (Today page, landing page demo) ---

type monthDay struct {
	Blank      bool // padding cell before the 1st / after the last day
	Day        int
	DateParam  string
	Future     bool
	IsToday    bool
	Selected   bool
	HasEntry   bool
	ScoreText  string
	Background string
}

type monthCalendar struct {
	Label       string
	Weekdays    []string
	Days        []monthDay
	MonthParam  string
	PrevParam   string
	NextParam   string
	HasPrev     bool
	HasNext     bool
	AverageText string
	Count       int
	Interactive bool
}

// buildMonthCalendar lays out a Monday-first month grid for the month starting
// at monthStart. selected highlights a day (zero time for none), and days after
// today are marked as future so they can't be picked.
func buildMonthCalendar(monthStart, today, selected time.Time, entries []models.MoodEntry, lang string, interactive bool) monthCalendar {
	scoreByDate := make(map[string]float64, len(entries))
	for _, e := range entries {
		scoreByDate[e.EntryDate.Format(dateParamLayout)] = e.Score
	}

	daysInMonth := monthStart.AddDate(0, 1, -1).Day()
	leading := (int(monthStart.Weekday()) + 6) % 7

	days := make([]monthDay, 0, 42)
	for i := 0; i < leading; i++ {
		days = append(days, monthDay{Blank: true})
	}

	var sum float64
	var count int
	for d := 1; d <= daysInMonth; d++ {
		date := time.Date(monthStart.Year(), monthStart.Month(), d, 0, 0, 0, 0, time.UTC)
		day := monthDay{
			Day:       d,
			DateParam: date.Format(dateParamLayout),
			Future:    date.After(today),
			IsToday:   date.Equal(today),
			Selected:  !selected.IsZero() && date.Equal(selected),
		}
		if score, ok := scoreByDate[day.DateParam]; ok {
			day.HasEntry = true
			day.ScoreText = scoreLabel(score)
			day.Background = scoreGradient(score).Hex()
			sum += score
			count++
		}
		days = append(days, day)
	}
	for len(days)%7 != 0 {
		days = append(days, monthDay{Blank: true})
	}

	prev := monthStart.AddDate(0, -1, 0)
	next := monthStart.AddDate(0, 1, 0)

	return monthCalendar{
		Label:       i18n.FormatMonthYear(lang, monthStart),
		Weekdays:    i18n.WeekdayAbbrs(lang),
		Days:        days,
		MonthParam:  monthStart.Format(monthParamLayout),
		PrevParam:   prev.Format(monthParamLayout),
		NextParam:   next.Format(monthParamLayout),
		HasPrev:     prev.Year() >= today.Year()-maxYearsBack,
		HasNext:     !next.After(startOfMonth(today)),
		AverageText: formatAverage(sum/math.Max(float64(count), 1), count),
		Count:       count,
		Interactive: interactive,
	}
}

// --- Year grid (profile page) ---

type yearCell struct {
	Valid      bool
	Future     bool
	Clickable  bool
	HasEntry   bool
	ScoreText  string
	Background string
	DateParam  string
	Title      string
}

type yearRow struct {
	Day   int
	Cells []yearCell
}

type yearCalendar struct {
	Year          int
	MonthNames    []string
	Rows          []yearRow
	MonthAverages []string
}

// buildYearCalendar lays out a whole year as 31 rows (days) by 12 columns
// (months), the "year in pixels" view.
func buildYearCalendar(year int, today time.Time, entries []models.MoodEntry, editable bool, lang string) yearCalendar {
	scoreByDate := make(map[string]float64, len(entries))
	for _, e := range entries {
		scoreByDate[e.EntryDate.Format(dateParamLayout)] = e.Score
	}

	monthNames := i18n.MonthAbbrs(lang)

	rows := make([]yearRow, 31)
	monthSums := make([]float64, 12)
	monthCounts := make([]int, 12)

	for day := 1; day <= 31; day++ {
		cells := make([]yearCell, 12)
		for month := 0; month < 12; month++ {
			daysInMonth := time.Date(year, time.Month(month+2), 0, 0, 0, 0, 0, time.UTC).Day()
			if day > daysInMonth {
				cells[month] = yearCell{Valid: false}
				continue
			}

			date := time.Date(year, time.Month(month+1), day, 0, 0, 0, 0, time.UTC)
			cell := yearCell{
				Valid:     true,
				Future:    date.After(today),
				DateParam: date.Format(dateParamLayout),
				Clickable: editable && !date.After(today),
				Title:     i18n.FormatDate(lang, date),
			}

			if score, ok := scoreByDate[cell.DateParam]; ok {
				cell.HasEntry = true
				cell.ScoreText = scoreLabel(score)
				cell.Background = scoreGradient(score).Hex()
				cell.Title += " · " + fmt.Sprintf("%.1f", score)
				monthSums[month] += score
				monthCounts[month]++
			}

			cells[month] = cell
		}
		rows[day-1] = yearRow{Day: day, Cells: cells}
	}

	monthAverages := make([]string, 12)
	for m := 0; m < 12; m++ {
		if monthCounts[m] == 0 {
			monthAverages[m] = "–"
		} else {
			monthAverages[m] = fmt.Sprintf("%.1f", monthSums[m]/float64(monthCounts[m]))
		}
	}

	return yearCalendar{Year: year, MonthNames: monthNames, Rows: rows, MonthAverages: monthAverages}
}
