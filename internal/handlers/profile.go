package handlers

import (
	"fmt"
	"math"
	"time"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type ProfileHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
}

func NewProfileHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository) *ProfileHandler {
	return &ProfileHandler{moods: moods, friendships: friendships}
}

const heatmapWeeks = 53

type heatmapCell struct {
	Date     time.Time
	Score    float64
	Level    int
	HasEntry bool
	Future   bool
}

// ScoreText formats the cell's score with a single decimal place, e.g. "7.6".
func (c heatmapCell) ScoreText() string {
	return fmt.Sprintf("%.1f", c.Score)
}

func (h *ProfileHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	ctx := c.Context()
	today := todayDate()

	average, count, err := h.moods.Stats(ctx, userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	weekAvg, weekCount, err := h.moods.StatsBetween(ctx, userID, startOfWeek(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	monthAvg, monthCount, err := h.moods.StatsBetween(ctx, userID, startOfMonth(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	yearAvg, yearCount, err := h.moods.StatsBetween(ctx, userID, startOfYear(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	dates, err := h.moods.ListEntryDates(ctx, userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	rangeStart := today.AddDate(0, 0, -(heatmapWeeks*7 - 1))
	rangeStart = rangeStart.AddDate(0, 0, -int(rangeStart.Weekday()))

	entries, err := h.moods.ListByUserSince(ctx, userID, rangeStart)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Render("pages/profile", withNav(ctx, h.friendships, userID, fiber.Map{
		"AverageScore":  fmt.Sprintf("%.1f", average),
		"TotalEntries":  count,
		"LongestStreak": longestStreak(dates),
		"WeekAverage":   formatAverage(weekAvg, weekCount),
		"MonthAverage":  formatAverage(monthAvg, monthCount),
		"YearAverage":   formatAverage(yearAvg, yearCount),
		"Weeks":         buildHeatmap(rangeStart, today, entries),
	}), "layouts/base")
}

func formatAverage(average float64, count int) string {
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f", average)
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

func longestStreak(dates []time.Time) int {
	if len(dates) == 0 {
		return 0
	}

	longest, current := 1, 1
	for i := 1; i < len(dates); i++ {
		if dates[i].Sub(dates[i-1]) == 24*time.Hour {
			current++
		} else {
			current = 1
		}
		if current > longest {
			longest = current
		}
	}
	return longest
}

func buildHeatmap(start, today time.Time, entries []models.MoodEntry) [][]heatmapCell {
	scoreByDate := make(map[string]float64, len(entries))
	for _, e := range entries {
		scoreByDate[e.EntryDate.Format("2006-01-02")] = e.Score
	}

	totalDays := int(today.Sub(start).Hours()/24) + 1
	totalCells := ((totalDays + 6) / 7) * 7

	weeks := make([][]heatmapCell, 0, heatmapWeeks)
	var week []heatmapCell

	for i := 0; i < totalCells; i++ {
		date := start.AddDate(0, 0, i)
		cell := heatmapCell{Date: date, Future: date.After(today)}

		if score, ok := scoreByDate[date.Format("2006-01-02")]; ok {
			cell.Score = score
			cell.Level = clampLevel(int(math.Round(score)))
			cell.HasEntry = true
		}

		week = append(week, cell)
		if len(week) == 7 {
			weeks = append(weeks, week)
			week = nil
		}
	}

	return weeks
}

func clampLevel(level int) int {
	if level < 1 {
		return 1
	}
	if level > 10 {
		return 10
	}
	return level
}
