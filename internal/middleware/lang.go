package middleware

import (
	"strings"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/gofiber/fiber/v2"
)

const LangCookieName = "lang"
const langContextKey = "lang"

// LoadLang determines the request's language - a saved cookie preference
// wins, otherwise the browser's Accept-Language header is used (Turkish
// browsers default to Turkish, everyone else defaults to English) - and
// stores it for handlers/templates to read via CurrentLang.
func LoadLang() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(langContextKey, resolveLang(c))
		return c.Next()
	}
}

func resolveLang(c *fiber.Ctx) string {
	if cookie := c.Cookies(LangCookieName); i18n.Supported(cookie) {
		return cookie
	}

	header := c.Get("Accept-Language")
	first := strings.SplitN(header, ",", 2)[0]
	first = strings.SplitN(first, ";", 2)[0]
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(first, "-", 2)[0]))

	if base == "tr" {
		return "tr"
	}
	return i18n.Default
}

// CurrentLang reads the language stored by LoadLang, defaulting to English.
func CurrentLang(c *fiber.Ctx) string {
	if lang, ok := c.Locals(langContextKey).(string); ok {
		return lang
	}
	return i18n.Default
}
