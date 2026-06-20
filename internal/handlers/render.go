package handlers

import (
	"context"

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
