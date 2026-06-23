package handlers

import (
	"errors"

	"github.com/emiryoneyler/mymood/internal/i18n"
	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
)

// resolveFriendTarget looks up a user by username and ensures the viewer is
// allowed to see their data - either it's their own username, or they're
// accepted friends. The returned error, if non-nil, is ready to be returned
// directly from the calling handler.
func resolveFriendTarget(c *fiber.Ctx, users *repository.UserRepository, friendships *repository.FriendshipRepository, viewerID, username string) (*models.User, error) {
	target, err := users.GetByUsername(c.Context(), username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fiber.ErrNotFound
	}
	if err != nil {
		return nil, fiber.ErrInternalServerError
	}

	if target.ID == viewerID {
		return target, nil
	}

	friendship, err := friendships.GetBetween(c.Context(), viewerID, target.ID)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && friendship.Status != models.FriendshipAccepted) {
		lang := middleware.CurrentLang(c)
		return nil, fiber.NewError(fiber.StatusForbidden, i18n.T(lang, "friend_access.forbidden"))
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fiber.ErrInternalServerError
	}

	return target, nil
}
