package handlers

import (
	"errors"
	"fmt"
	"math"
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
	flash := fiber.Map{
		"Saved":   c.Query("saved") == "1",
		"Removed": c.Query("removed") == "1",
	}
	return h.renderProfile(c, userID, userID, "", true, flash)
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

	return h.renderProfile(c, viewerID, target.ID, target.Username, false, fiber.Map{})
}

func (h *ProfileHandler) renderProfile(c *fiber.Ctx, viewerID, targetID, username string, editable bool, flash fiber.Map) error {
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

	data := fiber.Map{
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
		"Legend":          buildLegend(),
		"YearDaysTracked": yearCount,
		"TrackingRate":    trackingRate,
	}
	for k, v := range flash {
		data[k] = v
	}

	return c.Render("pages/profile", withNav(ctx, h.friendships, viewerID, data), "layouts/base")
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

// bucketFor classifies a score into one of five named buckets, used to group
// the distribution breakdown and the legend (the calendar itself uses a
// continuous gradient via scoreColor, not these discrete buckets).
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

type rgbColor struct{ R, G, B int }

func (c rgbColor) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// scoreGradient maps a 1-10 score to a continuous, vivid color: red at the
// bottom, through orange and yellow, to green, ending in turquoise at the top.
// Stops are evenly spaced so every tenth of a point produces a visibly
// different shade (e.g. 7.0 vs 7.5). Cell text is always white (see
// app.css .cal-cell text-shadow) so contrast stays consistent across the
// whole gradient instead of flipping between black/white per cell.
var scoreGradientStops = []struct {
	Pos   float64
	Color rgbColor
}{
	{1.00, rgbColor{220, 38, 38}},   // vivid red
	{3.25, rgbColor{249, 115, 22}},  // vivid orange
	{5.50, rgbColor{250, 204, 21}},  // vivid yellow
	{7.75, rgbColor{34, 197, 94}},   // vivid green
	{10.00, rgbColor{34, 211, 238}}, // vivid turquoise
}

func scoreGradient(score float64) rgbColor {
	stops := scoreGradientStops
	if score <= stops[0].Pos {
		return stops[0].Color
	}
	for i := 1; i < len(stops); i++ {
		if score <= stops[i].Pos {
			prev, next := stops[i-1], stops[i]
			t := (score - prev.Pos) / (next.Pos - prev.Pos)
			return rgbColor{
				R: lerp(prev.Color.R, next.Color.R, t),
				G: lerp(prev.Color.G, next.Color.G, t),
				B: lerp(prev.Color.B, next.Color.B, t),
			}
		}
	}
	return stops[len(stops)-1].Color
}

func lerp(a, b int, t float64) int {
	return a + int(math.Round(float64(b-a)*t))
}

type calendarCell struct {
	Valid      bool
	Clickable  bool
	HasEntry   bool
	ScoreText  string
	Background string
	DateParam  string
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
				cell.Background = scoreGradient(score).Hex()
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

// bucketRepresentative is a representative score for each named bucket, used
// to pick a single swatch color (from the same gradient the calendar uses)
// for the distribution and legend panels.
var bucketRepresentative = map[string]float64{
	"excellent": 9.5,
	"great":     8.0,
	"good":      6.0,
	"low":       4.0,
	"poor":      2.0,
}

var bucketOrder = []string{"excellent", "great", "good", "low", "poor"}

var bucketLabels = map[string]string{
	"excellent": "Mükemmel",
	"great":     "Harika",
	"good":      "İyi",
	"low":       "Düşük",
	"poor":      "Kötü",
}

type bucketStat struct {
	Label string
	Color string
	Count int
	Pct   int
}

func buildDistribution(entries []models.MoodEntry) []bucketStat {
	counts := map[string]int{}
	for _, e := range entries {
		_, class := bucketFor(e.Score)
		counts[class]++
	}

	total := len(entries)
	stats := make([]bucketStat, 0, len(bucketOrder))
	for _, class := range bucketOrder {
		pct := 0
		if total > 0 {
			pct = int(float64(counts[class]) / float64(total) * 100)
		}
		stats = append(stats, bucketStat{
			Label: bucketLabels[class],
			Color: scoreGradient(bucketRepresentative[class]).Hex(),
			Count: counts[class],
			Pct:   pct,
		})
	}

	return stats
}

type legendItem struct {
	Range string
	Label string
	Color string
}

func buildLegend() []legendItem {
	ranges := map[string]string{
		"excellent": "9-10",
		"great":     "7-8.9",
		"good":      "5-6.9",
		"low":       "3-4.9",
		"poor":      "1-2.9",
	}

	items := make([]legendItem, 0, len(bucketOrder))
	for _, class := range bucketOrder {
		items = append(items, legendItem{
			Range: ranges[class],
			Label: bucketLabels[class],
			Color: scoreGradient(bucketRepresentative[class]).Hex(),
		})
	}
	return items
}
