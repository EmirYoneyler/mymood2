package handlers

import (
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type MonthsHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
}

func NewMonthsHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository) *MonthsHandler {
	return &MonthsHandler{moods: moods, friendships: friendships}
}

var turkishMonthNames = []string{
	"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
	"Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık",
}

type monthRow struct {
	Label       string
	Year        int
	AverageText string
	Count       int
}

// Show lists every calendar month the user has logged a mood in, newest first,
// each linking back to that year's calendar on the profile page.
func (h *MonthsHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	summaries, err := h.moods.MonthlyBreakdown(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	rows := make([]monthRow, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, monthRow{
			Label:       turkishMonthNames[int(s.Month.Month())-1],
			Year:        s.Month.Year(),
			AverageText: s.AverageText(),
			Count:       s.Count,
		})
	}

	return c.Render("pages/months", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Months": rows,
	}), "layouts/base")
}
