package handlers

import (
	"github.com/emiryoneyler/mymood/internal/middleware"
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

func (h *FeedHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	friendIDs, err := h.friendships.ListFriendIDs(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	entries, err := h.moods.ListFeedForUserIDs(c.Context(), friendIDs, 50)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Render("pages/feed", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Entries":     entries,
		"HasFriends":  len(friendIDs) > 0,
		"FriendCount": len(friendIDs),
	}), "layouts/base")
}
