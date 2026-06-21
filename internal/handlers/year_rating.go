package handlers

import (
	"fmt"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type YearRatingHandler struct {
	ratings  *repository.YearRatingRepository
	validate *validator.Validate
}

func NewYearRatingHandler(ratings *repository.YearRatingRepository) *YearRatingHandler {
	return &YearRatingHandler{ratings: ratings, validate: validator.New()}
}

type yearRatingForm struct {
	Year  int     `form:"year" validate:"required,min=1900,max=2200"`
	Score float64 `form:"score" validate:"required,min=1,max=10"`
	Note  string  `form:"note" validate:"max=280"`
}

// Submit creates or updates a holistic rating for a past or current year,
// independent of any daily mood entries (e.g. "2024 was an 8/10 for me").
func (h *YearRatingHandler) Submit(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form yearRatingForm
	if err := c.BodyParser(&form); err != nil {
		return c.Redirect("/profile")
	}

	if err := h.validate.Struct(form); err != nil {
		return c.Redirect(fmt.Sprintf("/profile?year=%d", form.Year))
	}

	var note *string
	if form.Note != "" {
		note = &form.Note
	}

	if _, err := h.ratings.Upsert(c.Context(), userID, form.Year, form.Score, note); err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Redirect(fmt.Sprintf("/profile?year=%d&saved=1", form.Year))
}

type yearRatingDeleteForm struct {
	Year int `form:"year" validate:"required,min=1900,max=2200"`
}

func (h *YearRatingHandler) Delete(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form yearRatingDeleteForm
	if err := c.BodyParser(&form); err != nil {
		return c.Redirect("/profile")
	}

	if err := h.ratings.Delete(c.Context(), userID, form.Year); err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Redirect(fmt.Sprintf("/profile?year=%d&removed=1", form.Year))
}
