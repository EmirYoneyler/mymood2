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
	moods    *repository.MoodRepository
	validate *validator.Validate
}

func NewMoodHandler(moods *repository.MoodRepository) *MoodHandler {
	return &MoodHandler{
		moods:    moods,
		validate: validator.New(),
	}
}

type moodForm struct {
	Score int    `form:"score" validate:"required,min=1,max=10"`
	Note  string `form:"note" validate:"max=280"`
}

func todayDate() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func (h *MoodHandler) ShowForm(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	today, err := h.moods.GetByUserAndDate(c.Context(), userID, todayDate())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return c.Render("pages/mood", fiber.Map{
		"Today":    today,
		"NoteText": noteText(today),
		"MoodList": moodScale(),
	}, "layouts/base")
}

func (h *MoodHandler) Submit(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form moodForm
	if err := c.BodyParser(&form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/mood", fiber.Map{
			"Error":    "Geçersiz form verisi.",
			"MoodList": moodScale(),
		}, "layouts/base")
	}

	if err := h.validate.Struct(form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/mood", fiber.Map{
			"Error":    "Puan 1-10 arasında olmalı, not en fazla 280 karakter olabilir.",
			"MoodList": moodScale(),
		}, "layouts/base")
	}

	var note *string
	if form.Note != "" {
		note = &form.Note
	}

	entry, err := h.moods.Upsert(c.Context(), userID, form.Score, models.MoodEmoji(form.Score), note, todayDate())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).Render("pages/mood", fiber.Map{
			"Error":    "Mood kaydedilemedi, lütfen tekrar dene.",
			"MoodList": moodScale(),
		}, "layouts/base")
	}

	return c.Render("pages/mood", fiber.Map{
		"Today":    entry,
		"NoteText": noteText(entry),
		"Saved":    true,
		"MoodList": moodScale(),
	}, "layouts/base")
}

func noteText(entry *models.MoodEntry) string {
	if entry == nil || entry.Note == nil {
		return ""
	}
	return *entry.Note
}

type moodOption struct {
	Score int
	Emoji string
}

func moodScale() []moodOption {
	options := make([]moodOption, 0, 10)
	for score := 1; score <= 10; score++ {
		options = append(options, moodOption{Score: score, Emoji: models.MoodEmoji(score)})
	}
	return options
}
