package middleware

import "github.com/gofiber/fiber/v2"

// RequirePremium gates a route behind an active mymood+ subscription.
// isPremium should resolve the current request's user to their premium status
// (e.g. via users.is_premium today, or a subscriptions table lookup once Faz 2
// billing is wired up). Not used by any route yet — this exists so premium
// features can be added later without touching the routing layer.
func RequirePremium(isPremium func(c *fiber.Ctx) (bool, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ok, err := isPremium(c)
		if err != nil {
			return fiber.ErrInternalServerError
		}
		if !ok {
			return c.Status(fiber.StatusPaymentRequired).SendString("Bu özellik mymood+ üyeliği gerektirir.")
		}
		return c.Next()
	}
}
