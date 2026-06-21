package handlers

import (
	"errors"
	"strings"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	users     *repository.UserRepository
	jwtSecret string
	isProd    bool
	validate  *validator.Validate
}

func NewAuthHandler(users *repository.UserRepository, jwtSecret string, isProd bool) *AuthHandler {
	return &AuthHandler{
		users:     users,
		jwtSecret: jwtSecret,
		isProd:    isProd,
		validate:  validator.New(),
	}
}

type registerForm struct {
	Username        string `form:"username" validate:"required,min=3,max=30,alphanum"`
	Email           string `form:"email" validate:"required,email"`
	Password        string `form:"password" validate:"required,min=8,max=72"`
	PasswordConfirm string `form:"password_confirm" validate:"required,eqfield=Password"`
}

type loginForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required"`
	Next     string `form:"next"`
}

// safeRedirectTarget only allows redirecting to a same-site path, never to an
// external URL, so a crafted next= value can't be used to redirect users
// elsewhere after login.
func safeRedirectTarget(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/feed"
	}
	return next
}

func (h *AuthHandler) ShowRegister(c *fiber.Ctx) error {
	return c.Render("pages/register", fiber.Map{}, "layouts/base")
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var form registerForm
	if err := c.BodyParser(&form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/register", fiber.Map{
			"Error": "Geçersiz form verisi.",
		}, "layouts/base")
	}

	if err := h.validate.Struct(form); err != nil {
		message := "Lütfen tüm alanları doğru şekilde doldur (kullanıcı adı 3-30 karakter, sadece harf/rakam, boşluksuz; şifre en az 8 karakter)."
		if fieldErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fe := range fieldErrors {
				if fe.Field() == "PasswordConfirm" {
					message = "Şifreler eşleşmiyor."
				}
			}
		}
		return c.Status(fiber.StatusBadRequest).Render("pages/register", fiber.Map{
			"Error":    message,
			"Username": form.Username,
			"Email":    form.Email,
		}, "layouts/base")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	user, err := h.users.Create(c.Context(), form.Username, form.Email, string(passwordHash))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/register", fiber.Map{
			"Error":    "Bu kullanıcı adı veya e-posta zaten kullanılıyor.",
			"Username": form.Username,
			"Email":    form.Email,
		}, "layouts/base")
	}

	if err := h.issueSession(c, user.ID); err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Redirect("/feed")
}

func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	return c.Render("pages/login", fiber.Map{
		"Next":         c.Query("next"),
		"AuthRequired": c.Query("reason") == "auth",
	}, "layouts/base")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var form loginForm
	if err := c.BodyParser(&form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/login", fiber.Map{
			"Error": "Geçersiz form verisi.",
		}, "layouts/base")
	}

	if err := h.validate.Struct(form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("pages/login", fiber.Map{
			"Error": "E-posta ve şifre gereklidir.",
			"Email": form.Email,
			"Next":  form.Next,
		}, "layouts/base")
	}

	user, err := h.users.GetByEmail(c.Context(), form.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return c.Status(fiber.StatusUnauthorized).Render("pages/login", fiber.Map{
			"Error": "E-posta veya şifre yanlış.",
			"Email": form.Email,
			"Next":  form.Next,
		}, "layouts/base")
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(form.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).Render("pages/login", fiber.Map{
			"Error": "E-posta veya şifre yanlış.",
			"Email": form.Email,
			"Next":  form.Next,
		}, "layouts/base")
	}

	if err := h.issueSession(c, user.ID); err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Redirect(safeRedirectTarget(form.Next))
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
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

func (h *AuthHandler) issueSession(c *fiber.Ctx, userID string) error {
	token, err := middleware.GenerateToken(userID, h.jwtSecret)
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   h.isProd,
		SameSite: fiber.CookieSameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	})

	return nil
}
