package handlers

import (
	"errors"
	"time"

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

	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return today
	}

	parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	if parsed.After(today) {
		return today
	}
	return parsed
}

func (h *MoodHandler) ShowForm(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	entryDate := parseEntryDate(c.Query("date"))

	entry, err := h.moods.GetByUserAndDate(c.Context(), userID, entryDate)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return c.Render("pages/mood", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Today":        entry,
		"NoteText":     noteText(entry),
		"SelectedDate": entryDate,
		"IsToday":      entryDate.Equal(todayDate()),
		"MaxDate":      todayDate().Format("2006-01-02"),
		"Saved":        c.Query("saved") == "1",
	}), "layouts/base")
}

func (h *MoodHandler) Submit(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form moodForm
	if err := c.BodyParser(&form); err != nil {
		return h.renderError(c, userID, "Geçersiz form verisi.", todayDate())
	}

	entryDate := parseEntryDate(form.EntryDate)

	if err := h.validate.Struct(form); err != nil {
		return h.renderError(c, userID, "Puan 1.0-10.0 arasında olmalı, not en fazla 280 karakter olabilir.", entryDate)
	}

	var note *string
	if form.Note != "" {
		note = &form.Note
	}

	_, err := h.moods.Upsert(c.Context(), userID, form.Score, models.MoodEmoji(form.Score), note, entryDate)
	if err != nil {
		return h.renderError(c, userID, "Mood kaydedilemedi, lütfen tekrar dene.", entryDate)
	}

	return c.Redirect("/mood?date=" + entryDate.Format("2006-01-02") + "&saved=1")
}

func (h *MoodHandler) renderError(c *fiber.Ctx, userID, message string, entryDate time.Time) error {
	return c.Status(fiber.StatusBadRequest).Render("pages/mood", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Error":        message,
		"SelectedDate": entryDate,
		"IsToday":      entryDate.Equal(todayDate()),
		"MaxDate":      todayDate().Format("2006-01-02"),
	}), "layouts/base")
}

func noteText(entry *models.MoodEntry) string {
	if entry == nil || entry.Note == nil {
		return ""
	}
	return *entry.Note
}
