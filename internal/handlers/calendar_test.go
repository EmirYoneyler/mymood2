package handlers

import (
	"testing"
	"time"

	"github.com/emiryoneyler/mymood/internal/models"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestScoreGradientEndsAndStops(t *testing.T) {
	cases := map[float64]string{
		0.5:  "#ff3b30", // clamps below 1
		1:    "#ff3b30",
		5.5:  "#ffd60a",
		6.6:  "#a3e635",
		10:   "#06d6c7",
		11.0: "#06d6c7", // clamps above 10
	}
	for score, want := range cases {
		if got := scoreGradient(score).Hex(); got != want {
			t.Errorf("scoreGradient(%v) = %s, want %s", score, got, want)
		}
	}
}

func TestMonthCalendarMarksNotes(t *testing.T) {
	note := "great day"
	empty := ""
	entries := []models.MoodEntry{
		{Score: 8.2, EntryDate: date(2026, 10, 3), Note: &note},
		{Score: 4.0, EntryDate: date(2026, 10, 5)},
		{Score: 6.0, EntryDate: date(2026, 10, 6), Note: &empty},
	}
	cal := buildMonthCalendar(date(2026, 10, 1), date(2026, 10, 14), date(2026, 10, 14), entries, "en", true)

	byDay := map[int]monthDay{}
	for _, d := range cal.Days {
		if !d.Blank {
			byDay[d.Day] = d
		}
	}
	if d := byDay[3]; !d.HasEntry || !d.HasNote || d.Title != note {
		t.Errorf("day 3 should have entry and note, got %+v", d)
	}
	if d := byDay[5]; !d.HasEntry || d.HasNote {
		t.Errorf("day 5 has a score but no note, got %+v", d)
	}
	if d := byDay[6]; d.HasNote {
		t.Errorf("an empty note must not show the icon, got %+v", d)
	}
	if len(cal.Days)%7 != 0 {
		t.Errorf("month grid should fill whole weeks, got %d cells", len(cal.Days))
	}
	if cal.Count != 3 || cal.AverageText != "6.1" {
		t.Errorf("summary = %d days, avg %s", cal.Count, cal.AverageText)
	}
}

func TestYearCalendarMarksNotes(t *testing.T) {
	note := "trip"
	entries := []models.MoodEntry{
		{Score: 9, EntryDate: date(2026, 2, 10), Note: &note},
		{Score: 3, EntryDate: date(2026, 2, 11)},
	}
	cal := buildYearCalendar(2026, date(2026, 10, 14), entries, true, "en")

	withNote := cal.Rows[9].Cells[1] // 10 Feb
	without := cal.Rows[10].Cells[1] // 11 Feb
	if !withNote.HasNote || withNote.Title != "10 Feb 2026 · 9.0 · trip" {
		t.Errorf("10 Feb: %+v", withNote)
	}
	if without.HasNote || !without.HasEntry {
		t.Errorf("11 Feb: %+v", without)
	}
	if cal.Rows[29].Cells[1].Valid { // 30 Feb doesn't exist
		t.Error("30 Feb should be invalid")
	}
}
