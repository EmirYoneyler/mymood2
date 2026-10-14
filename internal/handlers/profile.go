package handlers

import (
	"strconv"
	"time"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type ProfileHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
	users       *repository.UserRepository
}

func NewProfileHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository, users *repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{moods: moods, friendships: friendships, users: users}
}

// Show renders the logged-in user's own profile.
func (h *ProfileHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return h.renderProfile(c, userID, userID, user.Username, true)
}

// ShowFriend renders another user's profile, restricted to accepted friends.
func (h *ProfileHandler) ShowFriend(c *fiber.Ctx) error {
	viewerID, _ := middleware.UserIDFromContext(c)
	username := c.Params("username")

	target, err := resolveFriendTarget(c, h.users, h.friendships, viewerID, username)
	if err != nil {
		return err
	}

	if target.ID == viewerID {
		return c.Redirect("/profile")
	}

	return h.renderProfile(c, viewerID, target.ID, target.Username, false)
}

func (h *ProfileHandler) renderProfile(c *fiber.Ctx, viewerID, targetID, username string, editable bool) error {
	ctx := c.Context()
	lang := middleware.CurrentLang(c)
	today := todayDate()

	average, count, err := h.moods.Stats(ctx, targetID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	monthAvg, monthCount, err := h.moods.StatsBetween(ctx, targetID, startOfMonth(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	yearAvg, yearCount, err := h.moods.StatsBetween(ctx, targetID, startOfYear(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	currentYear := today.Year()
	viewYear := currentYear
	if y, parseErr := strconv.Atoi(c.Query("year")); parseErr == nil {
		viewYear = clampYear(y, currentYear-maxYearsBack, currentYear)
	}

	viewYearStart := time.Date(viewYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	viewYearEnd := time.Date(viewYear, time.December, 31, 0, 0, 0, 0, time.UTC)
	if viewYear == currentYear {
		viewYearEnd = today
	}

	viewYearEntries, err := h.moods.ListBetween(ctx, targetID, viewYearStart, viewYearEnd)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	viewYearAvg, viewYearCount := summarizeEntries(viewYearEntries)

	profilePrefix := "/profile"
	if !editable {
		profilePrefix = "/profile/" + username
	}

	data := fiber.Map{
		"Editable":        editable,
		"Username":        username,
		"ProfileLink":     profilePrefix,
		"MonthsLink":      profilePrefix + "/months",
		"YearsLink":       profilePrefix + "/years",
		"AverageScore":    formatAverage(average, count),
		"TotalEntries":    count,
		"MonthAverage":    formatAverage(monthAvg, monthCount),
		"YearAverage":     formatAverage(yearAvg, yearCount),
		"Calendar":        buildYearCalendar(viewYear, today, viewYearEntries, editable, lang),
		"ViewYear":        viewYear,
		"PrevYear":        viewYear - 1,
		"NextYear":        viewYear + 1,
		"HasNextYear":     viewYear < currentYear,
		"CanGoBack":       viewYear > currentYear-maxYearsBack,
		"ViewYearAverage": formatAverage(viewYearAvg, viewYearCount),
		"YearDaysTracked": viewYearCount,
	}

	return renderPage(c, "pages/profile", withNav(c, h.friendships, viewerID, "profile", data), "layouts/base")
}
