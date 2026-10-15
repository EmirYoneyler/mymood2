package handlers

import (
	"testing"
	"time"
)

func TestLocalTodayUsesVisitorTimezone(t *testing.T) {
	istanbul, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		now  time.Time
		loc  *time.Location
		want time.Time
	}{
		// 01:30 in Istanbul on 15 Oct is still 14 Oct in UTC: the bug this fixes.
		{"after midnight in Istanbul", time.Date(2026, 10, 14, 22, 30, 0, 0, time.UTC), istanbul, date(2026, 10, 15)},
		{"evening in Istanbul", time.Date(2026, 10, 14, 20, 59, 0, 0, time.UTC), istanbul, date(2026, 10, 14)},
		{"exactly midnight in Istanbul", time.Date(2026, 10, 14, 21, 0, 0, 0, time.UTC), istanbul, date(2026, 10, 15)},
		{"New Year's Eve in Istanbul", time.Date(2026, 12, 31, 21, 15, 0, 0, time.UTC), istanbul, date(2027, 1, 1)},
		{"behind UTC", time.Date(2026, 10, 15, 2, 0, 0, 0, time.UTC), newYork, date(2026, 10, 14)},
		{"UTC", time.Date(2026, 10, 14, 23, 59, 0, 0, time.UTC), time.UTC, date(2026, 10, 14)},
	}
	for _, tc := range cases {
		got := localToday(tc.now, tc.loc)
		if !got.Equal(tc.want) || got.Location() != time.UTC {
			t.Errorf("%s: got %v, want %v (UTC midnight)", tc.name, got, tc.want)
		}
	}
}

func TestParseEntryDate(t *testing.T) {
	today := date(2026, 10, 15)
	cases := map[string]time.Time{
		"":           today,
		"2026-10-15": today,
		"2026-10-01": date(2026, 10, 1),
		"2026-10-16": today, // future days fall back to today
		"garbage":    today,
	}
	for raw, want := range cases {
		if got := parseEntryDate(raw, today); !got.Equal(want) {
			t.Errorf("parseEntryDate(%q) = %v, want %v", raw, got, want)
		}
	}
}
