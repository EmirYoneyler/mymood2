package handlers

import (
	"strings"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// SetLanguage stores the chosen language in a cookie and sends the visitor
// back to the page they were on (passed as ?next=, since helmet's
// no-referrer policy means there's no Referer header to go back to).
func SetLanguage(c *fiber.Ctx) error {
	code := c.Params("code")
	if i18n.Supported(code) {
		c.Cookie(&fiber.Cookie{
			Name:     middleware.LangCookieName,
			Value:    code,
			Path:     "/",
			MaxAge:   365 * 24 * 60 * 60,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}

	next := c.Query("next")
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") || strings.HasPrefix(next, "/lang/") {
		next = "/"
	}
	return c.Redirect(next)
}
