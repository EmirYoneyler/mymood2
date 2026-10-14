package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type MoodHandler struct {
	moods       *repository.MoodRepository
	friendships *repository.FriendshipRepository
	validate    *validator.Validate
}

func NewMoodHandler(moods *repository.MoodRepository, friendships *repository.FriendshipRepository) *MoodHandler {
	return &MoodHandler{
		moods:       moods,
		friendships: friendships,
		validate:    validator.New(),
	}
}

type moodForm struct {
	Score     float64 `form:"score" validate:"required,min=1,max=10"`
	Note      string  `form:"note" validate:"max=280"`
	EntryDate string  `form:"entry_date"`
}

// defaultScore is where the score picker starts for a day with no entry yet.
const defaultScore = 5.0

type deleteForm struct {
	EntryDate string `form:"entry_date"`
}

func todayDate() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// parseEntryDate parses a "2006-01-02" date string, falling back to today
// if it's missing, malformed, or in the future (you can't rate a day that
// hasn't happened yet).
func parseEntryDate(raw string) time.Time {
	today := todayDate()
	if raw == "" {
		return today
	}

	parsed, err := time.Parse(dateParamLayout, raw)
	if err != nil {
		return today
	}

	parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	if parsed.After(today) {
		return today
	}
	return parsed
}

// parseMonth parses a "2006-01" month string, falling back to the month of
// fallback if it's missing, malformed, in the future or too far back.
func parseMonth(raw string, fallback, today time.Time) time.Time {
	month := startOfMonth(fallback)
	if raw == "" {
		return month
	}
	parsed, err := time.Parse(monthParamLayout, raw)
	if err != nil {
		return month
	}
	parsed = startOfMonth(parsed)
	if parsed.After(startOfMonth(today)) || parsed.Year() < today.Year()-maxYearsBack {
		return month
	}
	return parsed
}

// ShowForm renders the Today page: the score form for the selected day (today
// by default) right at the top, and a month calendar underneath for jumping
// to any other day.
func (h *MoodHandler) ShowForm(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	entryDate := parseEntryDate(c.Query("date"))
	month := parseMonth(c.Query("month"), entryDate, todayDate())

	entry, err := h.moods.GetByUserAndDate(c.Context(), userID, entryDate)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	score := defaultScore
	if entry != nil {
		score = entry.Score
	}

	return h.renderForm(c, userID, entryDate, month, score, fiber.Map{
		"Entry":    entry,
		"NoteText": noteText(entry),
		"Saved":    c.Query("saved") == "1",
		"Removed":  c.Query("removed") == "1",
	})
}

func (h *MoodHandler) Submit(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	lang := middleware.CurrentLang(c)

	var form moodForm
	if err := c.BodyParser(&form); err != nil {
		return h.renderError(c, userID, i18n.T(lang, "mood.error_invalid_form"), parseEntryDate(c.FormValue("entry_date")), form)
	}

	entryDate := parseEntryDate(form.EntryDate)

	if err := h.validate.Struct(form); err != nil {
		return h.renderError(c, userID, i18n.T(lang, "mood.error_validation"), entryDate, form)
	}

	var note *string
	if form.Note != "" {
		note = &form.Note
	}

	_, err := h.moods.Upsert(c.Context(), userID, form.Score, models.MoodEmoji(form.Score), note, entryDate)
	if err != nil {
		return h.renderError(c, userID, i18n.T(lang, "mood.error_save_failed"), entryDate, form)
	}

	return c.Redirect("/mood?date=" + entryDate.Format(dateParamLayout) + "&saved=1")
}

// Delete removes the mood entry for the given date, so the user can clear a
// wrongly-entered score and leave the day blank again.
func (h *MoodHandler) Delete(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form deleteForm
	if err := c.BodyParser(&form); err != nil {
		return c.Redirect("/mood")
	}

	entryDate := parseEntryDate(form.EntryDate)

	if err := h.moods.Delete(c.Context(), userID, entryDate); err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Redirect("/mood?date=" + entryDate.Format(dateParamLayout) + "&removed=1")
}

// renderError re-renders the form with the user's input preserved, so a
// validation problem never throws away what they typed.
func (h *MoodHandler) renderError(c *fiber.Ctx, userID, message string, entryDate time.Time, form moodForm) error {
	entry, err := h.moods.GetByUserAndDate(c.Context(), userID, entryDate)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		entry = nil
	}

	score := defaultScore
	if form.Score >= 1 && form.Score <= 10 {
		score = form.Score
	} else if entry != nil {
		score = entry.Score
	}

	c.Status(fiber.StatusBadRequest)
	return h.renderForm(c, userID, entryDate, startOfMonth(entryDate), score, fiber.Map{
		"Error":    message,
		"Entry":    entry,
		"NoteText": form.Note,
	})
}

// renderForm renders the Today page for entryDate, with the month calendar
// showing month and the score picker starting at score.
func (h *MoodHandler) renderForm(c *fiber.Ctx, userID string, entryDate, month time.Time, score float64, data fiber.Map) error {
	lang := middleware.CurrentLang(c)
	today := todayDate()

	monthEntries, err := h.moods.ListBetween(c.Context(), userID, month, month.AddDate(0, 1, -1))
	if err != nil {
		return fiber.ErrInternalServerError
	}

	isToday := entryDate.Equal(today)
	prevDay := entryDate.AddDate(0, 0, -1)
	nextDay := entryDate.AddDate(0, 0, 1)

	data["SelectedDate"] = entryDate.Format(dateParamLayout)
	data["SelectedDateText"] = i18n.FormatLongDate(lang, entryDate, entryDate.Year() != today.Year())
	data["IsToday"] = isToday
	data["IsYesterday"] = entryDate.Equal(today.AddDate(0, 0, -1))
	data["PrevDate"] = prevDay.Format(dateParamLayout)
	data["HasPrevDate"] = prevDay.Year() >= today.Year()-maxYearsBack
	data["NextDate"] = nextDay.Format(dateParamLayout)
	data["HasNextDate"] = !nextDay.After(today)
	data["ScoreValue"] = fmt.Sprintf("%.1f", score)
	data["ScoreColor"] = scoreGradient(score).Hex()
	data["ScoreBucket"] = "bucket." + bucketFor(score)
	data["Calendar"] = buildMonthCalendar(month, today, entryDate, monthEntries, lang, true)

	return renderPage(c, "pages/mood", withNav(c, h.friendships, userID, "today", data), "layouts/base")
}

func noteText(entry *models.MoodEntry) string {
	if entry == nil || entry.Note == nil {
		return ""
	}
	return *entry.Note
}
