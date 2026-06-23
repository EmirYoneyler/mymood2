package handlers

import (
	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	users       *repository.UserRepository
	friendships *repository.FriendshipRepository
	isProd      bool
}

func NewSettingsHandler(users *repository.UserRepository, friendships *repository.FriendshipRepository, isProd bool) *SettingsHandler {
	return &SettingsHandler{users: users, friendships: friendships, isProd: isProd}
}

func (h *SettingsHandler) Show(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return renderPage(c, "pages/settings", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Username": user.Username,
		"Email":    user.Email,
	}), "layouts/base")
}

type deleteAccountForm struct {
	Password string `form:"password" validate:"required"`
}

func (h *SettingsHandler) DeleteAccount(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	lang := middleware.CurrentLang(c)

	var form deleteAccountForm
	if err := c.BodyParser(&form); err != nil {
		return h.renderSettingsError(c, userID, i18n.T(lang, "settings.error_invalid_form"))
	}

	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(form.Password)); err != nil {
		return h.renderSettingsError(c, userID, i18n.T(lang, "settings.error_wrong_password"))
	}

	if err := h.users.Delete(c.Context(), userID); err != nil {
		return fiber.ErrInternalServerError
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   h.isProd,
		SameSite: fiber.CookieSameSiteStrictMode,
		MaxAge:   -1,
	})

	return c.Redirect("/login")
}

func (h *SettingsHandler) renderSettingsError(c *fiber.Ctx, userID, message string) error {
	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	c.Status(fiber.StatusBadRequest)
	return renderPage(c, "pages/settings", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Username": user.Username,
		"Email":    user.Email,
		"Error":    message,
	}), "layouts/base")
}
