package handlers

import (
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type MonthsHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
	users       *repository.UserRepository
}

func NewMonthsHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository, users *repository.UserRepository) *MonthsHandler {
	return &MonthsHandler{moods: moods, friendships: friendships, users: users}
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

// Show lists every calendar month the logged-in user has logged a mood in.
func (h *MonthsHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	return h.render(c, userID, userID, "")
}

// ShowFriend lists another user's monthly breakdown, restricted to accepted friends.
func (h *MonthsHandler) ShowFriend(c *fiber.Ctx) error {
	viewerID, _ := middleware.UserIDFromContext(c)
	username := c.Params("username")

	target, err := resolveFriendTarget(c, h.users, h.friendships, viewerID, username)
	if err != nil {
		return err
	}

	return h.render(c, viewerID, target.ID, target.Username)
}

// render lists every calendar month targetID has logged a mood in, newest
// first, each linking back to that year's calendar on the profile page.
func (h *MonthsHandler) render(c *fiber.Ctx, viewerID, targetID, username string) error {
	summaries, err := h.moods.MonthlyBreakdown(c.Context(), targetID)
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

	profileLink := "/profile"
	if username != "" {
		profileLink = "/profile/" + username
	}

	return c.Render("pages/months", withNav(c.Context(), h.friendships, viewerID, fiber.Map{
		"Months":      rows,
		"Username":    username,
		"ProfileLink": profileLink,
	}), "layouts/base")
}
