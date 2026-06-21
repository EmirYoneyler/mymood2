package middleware

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
)

const ContextUserIDKey = "userID"

// RequireAuth rejects the request if no valid session cookie is present,
// sending the visitor to /login with the original URL preserved so they can
// be sent back there after signing in.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := userIDFromCookie(c, jwtSecret)
		if !ok {
			next := url.QueryEscape(c.OriginalURL())
			return c.Redirect("/login?reason=auth&next=" + next)
		}
		c.Locals(ContextUserIDKey, userID)
		return c.Next()
	}
}

// LoadSession attaches the user ID to the context if a valid session exists,
// without rejecting the request. Useful for pages that render differently
// for logged-in vs anonymous users.
func LoadSession(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if userID, ok := userIDFromCookie(c, jwtSecret); ok {
			c.Locals(ContextUserIDKey, userID)
		}
		return c.Next()
	}
}

func userIDFromCookie(c *fiber.Ctx, jwtSecret string) (string, bool) {
	cookie := c.Cookies(SessionCookieName)
	if cookie == "" {
		return "", false
	}

	userID, err := ParseToken(cookie, jwtSecret)
	if err != nil {
		return "", false
	}

	return userID, true
}

// UserIDFromContext reads the authenticated user ID set by RequireAuth/LoadSession.
func UserIDFromContext(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals(ContextUserIDKey).(string)
	return userID, ok
}
