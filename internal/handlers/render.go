package handlers

import (
	"context"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

// withNav adds the shared navigation bar data (pending friend request badge)
// to a page's render data.
func withNav(ctx context.Context, friendships *repository.FriendshipRepository, userID string, data fiber.Map) fiber.Map {
	count, _ := friendships.CountPendingIncoming(ctx, userID)
	data["ShowNav"] = true
	data["PendingCount"] = count
	return data
}

// renderPage injects the current request's translator ("T", callable from
// templates as {{call .T "some.key"}}) and language code before rendering,
// so every page - with or without nav - can be localized. Use this instead
// of calling c.Render directly.
func renderPage(c *fiber.Ctx, view string, data fiber.Map, layout string) error {
	if data == nil {
		data = fiber.Map{}
	}
	lang := middleware.CurrentLang(c)
	data["T"] = i18n.Translator(lang)
	data["CurrentLang"] = lang
	return c.Render(view, data, layout)
}
