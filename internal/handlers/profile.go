package handlers

import (
	"errors"
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
	users       *repository.UserRepository
}

func NewProfileHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository, users *repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{moods: moods, friendships: friendships, users: users}
}

// inactivityLimit is how long a user can go without logging a mood before
// their current streak resets to zero.
const inactivityLimit = 36 * time.Hour

// Show renders the logged-in user's own profile.
func (h *ProfileHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	return h.renderProfile(c, userID, userID, "", true)
}

// ShowFriend renders another user's profile, restricted to accepted friends.
func (h *ProfileHandler) ShowFriend(c *fiber.Ctx) error {
	viewerID, _ := middleware.UserIDFromContext(c)
	username := c.Params("username")

	target, err := h.users.GetByUsername(c.Context(), username)
	if errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}

	if target.ID == viewerID {
		return c.Redirect("/profile")
	}

	friendship, err := h.friendships.GetBetween(c.Context(), viewerID, target.ID)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && friendship.Status != models.FriendshipAccepted) {
		return c.Status(fiber.StatusForbidden).SendString("Bu profili görmek için arkadaş olmanız gerekiyor.")
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return h.renderProfile(c, viewerID, target.ID, target.Username, false)
}

func (h *ProfileHandler) renderProfile(c *fiber.Ctx, viewerID, targetID, username string, editable bool) error {
	ctx := c.Context()
	today := todayDate()

	average, count, err := h.moods.Stats(ctx, targetID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	weekAvg, weekCount, err := h.moods.StatsBetween(ctx, targetID, startOfWeek(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	monthAvg, monthCount, err := h.moods.StatsBetween(ctx, targetID, startOfMonth(today), today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	yearStart := startOfYear(today)
	yearAvg, yearCount, err := h.moods.StatsBetween(ctx, targetID, yearStart, today)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	dates, err := h.moods.ListEntryDates(ctx, targetID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	lastActivity, hasActivity, err := h.moods.LastActivityAt(ctx, targetID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	yearEntries, err := h.moods.ListByUserSince(ctx, targetID, yearStart)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	daysElapsedInYear := int(today.Sub(yearStart).Hours()/24) + 1
	trackingRate := 0
	if daysElapsedInYear > 0 {
		trackingRate = int(float64(yearCount) / float64(daysElapsedInYear) * 100)
	}

	return c.Render("pages/profile", withNav(ctx, h.friendships, viewerID, fiber.Map{
		"Editable":        editable,
		"Username":        username,
		"AverageScore":    fmt.Sprintf("%.1f", average),
		"TotalEntries":    count,
		"CurrentStreak":   currentStreak(dates, lastActivity, hasActivity, today),
		"LongestStreak":   longestStreak(dates),
		"WeekAverage":     formatAverage(weekAvg, weekCount),
		"MonthAverage":    formatAverage(monthAvg, monthCount),
		"YearAverage":     formatAverage(yearAvg, yearCount),
		"Calendar":        buildYearCalendar(today.Year(), today, yearEntries, editable),
		"Distribution":    buildDistribution(yearEntries),
		"YearDaysTracked": yearCount,
		"TrackingRate":    trackingRate,
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

// longestStreak returns the longest run of consecutive logged days, ever.
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

// currentStreak returns the run of consecutive logged days ending at the most
// recent entry, but resets to zero if the user hasn't logged anything in the
// last 36 hours (inactivityLimit) - so the streak actually requires showing
// up, not just having logged in a row at some point in the past.
func currentStreak(dates []time.Time, lastActivity time.Time, hasActivity bool, now time.Time) int {
	if !hasActivity || len(dates) == 0 {
		return 0
	}

	if now.Sub(lastActivity) > inactivityLimit {
		return 0
	}

	streak := 1
	for i := len(dates) - 1; i > 0; i-- {
		if dates[i].Sub(dates[i-1]) == 24*time.Hour {
			streak++
		} else {
			break
		}
	}
	return streak
}

// bucketFor classifies a score into one of five buckets used for calendar
// coloring and the distribution breakdown.
func bucketFor(score float64) (label, class string) {
	switch {
	case score >= 9:
		return "Mükemmel", "excellent"
	case score >= 7:
		return "Harika", "great"
	case score >= 5:
		return "İyi", "good"
	case score >= 3:
		return "Düşük", "low"
	default:
		return "Kötü", "poor"
	}
}

type calendarCell struct {
	Valid     bool
	Clickable bool
	HasEntry  bool
	ScoreText string
	Bucket    string
	DateParam string
}

type calendarRow struct {
	Day   int
	Cells []calendarCell
}

type yearCalendar struct {
	Year          int
	MonthNames    []string
	Rows          []calendarRow
	MonthAverages []string
}

func buildYearCalendar(year int, today time.Time, entries []models.MoodEntry, editable bool) yearCalendar {
	scoreByDate := make(map[string]float64, len(entries))
	for _, e := range entries {
		scoreByDate[e.EntryDate.Format("2006-01-02")] = e.Score
	}

	monthNames := []string{"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}

	rows := make([]calendarRow, 31)
	monthSums := make([]float64, 12)
	monthCounts := make([]int, 12)

	for day := 1; day <= 31; day++ {
		cells := make([]calendarCell, 12)
		for month := 0; month < 12; month++ {
			daysInMonth := time.Date(year, time.Month(month+2), 0, 0, 0, 0, 0, time.UTC).Day()
			if day > daysInMonth {
				cells[month] = calendarCell{Valid: false}
				continue
			}

			date := time.Date(year, time.Month(month+1), day, 0, 0, 0, 0, time.UTC)
			cell := calendarCell{
				Valid:     true,
				DateParam: date.Format("2006-01-02"),
				Clickable: editable && !date.After(today),
			}

			if score, ok := scoreByDate[cell.DateParam]; ok {
				cell.HasEntry = true
				cell.ScoreText = fmt.Sprintf("%.1f", score)
				_, cell.Bucket = bucketFor(score)
				monthSums[month] += score
				monthCounts[month]++
			}

			cells[month] = cell
		}
		rows[day-1] = calendarRow{Day: day, Cells: cells}
	}

	monthAverages := make([]string, 12)
	for m := 0; m < 12; m++ {
		if monthCounts[m] == 0 {
			monthAverages[m] = "—"
		} else {
			monthAverages[m] = fmt.Sprintf("%.1f", monthSums[m]/float64(monthCounts[m]))
		}
	}

	return yearCalendar{Year: year, MonthNames: monthNames, Rows: rows, MonthAverages: monthAverages}
}

type bucketStat struct {
	Label string
	Class string
	Count int
	Pct   int
}

func buildDistribution(entries []models.MoodEntry) []bucketStat {
	order := []string{"excellent", "great", "good", "low", "poor"}
	labels := map[string]string{
		"excellent": "Mükemmel",
		"great":     "Harika",
		"good":      "İyi",
		"low":       "Düşük",
		"poor":      "Kötü",
	}

	counts := map[string]int{}
	for _, e := range entries {
		_, class := bucketFor(e.Score)
		counts[class]++
	}

	total := len(entries)
	stats := make([]bucketStat, 0, len(order))
	for _, class := range order {
		pct := 0
		if total > 0 {
			pct = int(float64(counts[class]) / float64(total) * 100)
		}
		stats = append(stats, bucketStat{Label: labels[class], Class: class, Count: counts[class], Pct: pct})
	}

	return stats
}
