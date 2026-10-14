package handlers

import (
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type YearsHandler struct {
	moods       *repository.MoodRepository
	yearRatings *repository.YearRatingRepository
	friendships *repository.FriendshipRepository
	users       *repository.UserRepository
}

func NewYearsHandler(moods *repository.MoodRepository, yearRatings *repository.YearRatingRepository, friendships *repository.FriendshipRepository, users *repository.UserRepository) *YearsHandler {
	return &YearsHandler{moods: moods, yearRatings: yearRatings, friendships: friendships, users: users}
}

type yearListRow struct {
	Year      int
	IsCurrent bool
	Score     float64
	ScoreText string
	HasScore  bool
	NoteText  string
}

// Show lists the logged-in user's years.
func (h *YearsHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	return h.render(c, userID, userID, "", true)
}

// ShowFriend lists another user's years, read-only, restricted to accepted friends.
func (h *YearsHandler) ShowFriend(c *fiber.Ctx) error {
	viewerID, _ := middleware.UserIDFromContext(c)
	username := c.Params("username")

	target, err := resolveFriendTarget(c, h.users, h.friendships, viewerID, username)
	if err != nil {
		return err
	}

	return h.render(c, viewerID, target.ID, target.Username, false)
}

// render lists the current year (its live daily average, read-only) followed
// by every past year that has a holistic rating - useful for years never
// logged day-by-day but still remembered well. When viewing your own years,
// each rating can be edited or removed, and any past year can be rated.
func (h *YearsHandler) render(c *fiber.Ctx, viewerID, targetID, username string, editable bool) error {
	ctx := c.Context()
	today := todayDate()
	currentYear := today.Year()

	ratings, err := h.yearRatings.ListByUser(ctx, targetID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	avg, count, err := h.moods.StatsBetween(ctx, targetID, startOfYear(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	rows := make([]yearListRow, 0, len(ratings)+1)
	rows = append(rows, yearListRow{
		Year:      currentYear,
		IsCurrent: true,
		Score:     avg,
		ScoreText: formatAverage(avg, count),
		HasScore:  count > 0,
	})

	rated := make(map[int]bool, len(ratings))
	for _, r := range ratings { // already newest year first
		rated[r.Year] = true
		if r.Year >= currentYear {
			continue
		}
		rows = append(rows, yearListRow{
			Year:      r.Year,
			Score:     r.Score,
			ScoreText: r.ScoreText(),
			HasScore:  true,
			NoteText:  r.NoteText(),
		})
	}

	// Past years that can still be rated, for the "rate a year" picker.
	var unrated []int
	for year := currentYear - 1; year >= currentYear-maxYearsBack; year-- {
		if !rated[year] {
			unrated = append(unrated, year)
		}
	}

	profileLink := "/profile"
	if username != "" {
		profileLink = "/profile/" + username
	}

	return renderPage(c, "pages/years", withNav(c, h.friendships, viewerID, "profile", fiber.Map{
		"Years":        rows,
		"UnratedYears": unrated,
		"Editable":     editable,
		"Username":     username,
		"ProfileLink":  profileLink,
		"Saved":        c.Query("saved") == "1",
		"Removed":      c.Query("removed") == "1",
		"TitleKey":     "years.title",
	}), "layouts/base")
}
