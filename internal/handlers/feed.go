package handlers

import (
	"errors"
	"time"

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
	User models.User
	Mood models.MoodEntry
}

// feedGroup is one day's worth of friends' entries ("Today", "Yesterday", ...).
type feedGroup struct {
	Label string
	Rows  []feedRow
}

func (h *FeedHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	lang := middleware.CurrentLang(c)
	today := todayDate()

	friendIDs, err := h.friendships.ListFriendIDs(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	entries, err := h.moods.ListFeedForUserIDs(c.Context(), friendIDs, 50)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	_, err = h.moods.GetByUserAndDate(c.Context(), userID, today)
	ratedToday := err == nil
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return renderPage(c, "pages/feed", withNav(c, h.friendships, userID, "feed", fiber.Map{
		"Groups":     groupFeed(entries, today, lang),
		"HasFriends": len(friendIDs) > 0,
		"RatedToday": ratedToday,
	}), "layouts/base")
}

// groupFeed splits feed entries (already sorted newest first) into per-day groups.
func groupFeed(entries []models.FeedEntry, today time.Time, lang string) []feedGroup {
	t := i18n.Translator(lang)
	var groups []feedGroup
	var current time.Time

	for _, e := range entries {
		date := e.Mood.EntryDate
		if len(groups) == 0 || !date.Equal(current) {
			current = date
			groups = append(groups, feedGroup{Label: dayLabel(t, lang, date, today)})
		}
		last := &groups[len(groups)-1]
		last.Rows = append(last.Rows, feedRow{User: e.User, Mood: e.Mood})
	}
	return groups
}

func dayLabel(t func(string, ...any) string, lang string, date, today time.Time) string {
	switch {
	case date.Equal(today):
		return t("common.today")
	case date.Equal(today.AddDate(0, 0, -1)):
		return t("common.yesterday")
	case date.Year() == today.Year():
		return i18n.FormatLongDate(lang, date, false)
	default:
		return i18n.FormatLongDate(lang, date, true)
	}
}
