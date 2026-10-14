package handlers

import (
	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

// withNav adds the shared navigation data to a page's render data: the
// pending friend request badge, which section is active ("today", "feed",
// "friends", "profile" or "settings"), and the matching page title.
func withNav(c *fiber.Ctx, friendships *repository.FriendshipRepository, userID, section string, data fiber.Map) fiber.Map {
	count, _ := friendships.CountPendingIncoming(c.Context(), userID)
	data["ShowNav"] = true
	data["PendingCount"] = count
	data["Nav"] = section
	if _, ok := data["TitleKey"]; !ok {
		data["TitleKey"] = "nav." + section
	}
	return data
}

// pageTitleKeys gives pages rendered without withNav a <title>.
var pageTitleKeys = map[string]string{
	"pages/login":    "login.title",
	"pages/register": "register.title",
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
	data["CurrentPath"] = c.OriginalURL()
	if _, ok := data["TitleKey"]; !ok {
		data["TitleKey"] = pageTitleKeys[view]
	}
	return c.Render(view, data, layout)
}
