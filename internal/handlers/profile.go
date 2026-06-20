package handlers

import (
	"fmt"
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
	Score    int
	HasEntry bool
	Future   bool
}

func (h *ProfileHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	average, count, err := h.moods.Stats(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	dates, err := h.moods.ListEntryDates(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	today := todayDate()
	rangeStart := today.AddDate(0, 0, -(heatmapWeeks*7 - 1))
	rangeStart = rangeStart.AddDate(0, 0, -int(rangeStart.Weekday()))

	entries, err := h.moods.ListByUserSince(c.Context(), userID, rangeStart)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Render("pages/profile", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"AverageScore":  fmt.Sprintf("%.1f", average),
		"TotalEntries":  count,
		"LongestStreak": longestStreak(dates),
		"Weeks":         buildHeatmap(rangeStart, today, entries),
	}), "layouts/base")
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
	scoreByDate := make(map[string]int, len(entries))
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
