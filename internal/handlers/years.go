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
}

func NewYearsHandler(moods *repository.MoodRepository, yearRatings *repository.YearRatingRepository, friendships *repository.FriendshipRepository) *YearsHandler {
	return &YearsHandler{moods: moods, yearRatings: yearRatings, friendships: friendships}
}

// defaultYearsBack is how far back the years list goes by default. It's
// extended further if an older rating already exists, so a year you already
// rated never silently falls off the list.
const defaultYearsBack = 30

type yearRow struct {
	Year      int
	IsCurrent bool
	ScoreText string
	HasRating bool
	NoteText  string
}

// Show lists every year from the current one back through a reasonable
// history, current year showing the live daily average (read-only), and past
// years showing/accepting a single holistic rating you enter directly -
// useful for years you never logged day-by-day but still remember well.
func (h *YearsHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	ctx := c.Context()
	today := todayDate()
	currentYear := today.Year()

	ratings, err := h.yearRatings.ListByUser(ctx, userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	ratingByYear := make(map[int]string, len(ratings))
	noteByYear := make(map[int]string, len(ratings))
	minYear := currentYear - defaultYearsBack
	for _, r := range ratings {
		ratingByYear[r.Year] = r.ScoreText()
		noteByYear[r.Year] = r.NoteText()
		if r.Year < minYear {
			minYear = r.Year
		}
	}

	avg, count, err := h.moods.StatsBetween(ctx, userID, startOfYear(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	rows := make([]yearRow, 0, currentYear-minYear+1)
	for year := currentYear; year >= minYear; year-- {
		if year == currentYear {
			rows = append(rows, yearRow{Year: year, IsCurrent: true, ScoreText: formatAverage(avg, count)})
			continue
		}

		score, hasRating := ratingByYear[year]
		row := yearRow{Year: year, ScoreText: "—"}
		if hasRating {
			row.ScoreText = score
			row.HasRating = true
			row.NoteText = noteByYear[year]
		}
		rows = append(rows, row)
	}

	return c.Render("pages/years", withNav(ctx, h.friendships, userID, fiber.Map{
		"Years":   rows,
		"Saved":   c.Query("saved") == "1",
		"Removed": c.Query("removed") == "1",
	}), "layouts/base")
}
