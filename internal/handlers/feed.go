package handlers

import (
	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type FeedHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
}

func NewFeedHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository) *FeedHandler {
	return &FeedHandler{moods: moods, friendships: friendships}
}

type feedRow struct {
	User     models.User
	Mood     models.MoodEntry
	DateText string
}

func (h *FeedHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	lang := middleware.CurrentLang(c)

	friendIDs, err := h.friendships.ListFriendIDs(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	entries, err := h.moods.ListFeedForUserIDs(c.Context(), friendIDs, 50)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	rows := make([]feedRow, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, feedRow{
			User:     e.User,
			Mood:     e.Mood,
			DateText: i18n.FormatDate(lang, e.Mood.EntryDate),
		})
	}

	return renderPage(c, "pages/feed", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Entries":     rows,
		"HasFriends":  len(friendIDs) > 0,
		"FriendCount": len(friendIDs),
	}), "layouts/base")
}
