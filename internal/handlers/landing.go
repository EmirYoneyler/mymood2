package handlers

import (
	"math"
	"time"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/gofiber/fiber/v2"
)

// Landing renders the public home page for signed-out visitors; signed-in
// users go straight to today's entry.
func Landing(c *fiber.Ctx) error {
	if _, ok := middleware.UserIDFromContext(c); ok {
		return c.Redirect("/mood")
	}

	lang := middleware.CurrentLang(c)
	today := todayDate()
	month := startOfMonth(today).AddDate(0, -1, 0)

	return renderPage(c, "pages/landing", fiber.Map{
		"DemoCalendar": buildMonthCalendar(month, today, time.Time{}, demoEntries(month), lang, false),
	}, "layouts/base")
}

// demoEntries fills a month with plausible, deterministic sample scores for
// the landing page preview (a few days left blank, like a real month).
func demoEntries(month time.Time) []models.MoodEntry {
	days := month.AddDate(0, 1, -1).Day()
	entries := make([]models.MoodEntry, 0, days)
	for d := 1; d <= days; d++ {
		if d%9 == 4 {
			continue
		}
		wave := math.Sin(float64(d)*0.55) * 1.8
		jitter := float64((d*37)%11)/10 - 0.5
		score := math.Round((6.8+wave+jitter)*10) / 10
		score = math.Max(1, math.Min(10, score))
		entries = append(entries, models.MoodEntry{
			Score:     score,
			EntryDate: time.Date(month.Year(), month.Month(), d, 0, 0, 0, 0, time.UTC),
		})
	}
	return entries
}
