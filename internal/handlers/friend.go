package handlers

import (
	"errors"

	"github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type FriendHandler struct {
	users       *repository.UserRepository
	friendships *repository.FriendshipRepository
	validate    *validator.Validate
}

func NewFriendHandler(users *repository.UserRepository, friendships *repository.FriendshipRepository) *FriendHandler {
	return &FriendHandler{
		users:       users,
		friendships: friendships,
		validate:    validator.New(),
	}
}

type friendRequestForm struct {
	Username string `form:"username" validate:"required,min=3,max=30"`
}

func (h *FriendHandler) ShowFriends(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	query := c.Query("q")

	friends, err := h.friendships.ListFriends(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	pending, err := h.friendships.ListPendingIncoming(c.Context(), userID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	var results []models.User
	if query != "" {
		results, err = h.users.SearchByUsername(c.Context(), query, userID)
		if err != nil {
			return fiber.ErrInternalServerError
		}
	}

	return renderPage(c, "pages/friends", withNav(c.Context(), h.friendships, userID, fiber.Map{
		"Friends": friends,
		"Pending": pending,
		"Query":   query,
		"Results": results,
	}), "layouts/base")
}

func (h *FriendHandler) SendRequest(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)

	var form friendRequestForm
	if err := c.BodyParser(&form); err != nil || h.validate.Struct(form) != nil {
		return c.Redirect("/friends")
	}

	target, err := h.users.GetByUsername(c.Context(), form.Username)
	if errors.Is(err, repository.ErrNotFound) {
		return c.Redirect("/friends?q=" + form.Username)
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}

	if target.ID == userID {
		return c.Redirect("/friends")
	}

	existing, err := h.friendships.GetBetween(c.Context(), userID, target.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	switch {
	case errors.Is(err, repository.ErrNotFound):
		if _, err := h.friendships.Create(c.Context(), userID, target.ID); err != nil {
			return fiber.ErrInternalServerError
		}
	case existing.Status == models.FriendshipAccepted:
		// already friends, nothing to do
	case existing.RequesterID == target.ID:
		// they already requested us, accept it now
		if err := h.friendships.Accept(c.Context(), existing.ID, userID); err != nil {
			return fiber.ErrInternalServerError
		}
	}

	return c.Redirect("/friends")
}

func (h *FriendHandler) AcceptRequest(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	friendshipID := c.Params("id")

	if err := h.friendships.Accept(c.Context(), friendshipID, userID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return c.Redirect("/friends")
}

func (h *FriendHandler) RejectRequest(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	friendshipID := c.Params("id")

	if err := h.friendships.Reject(c.Context(), friendshipID, userID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return c.Redirect("/friends")
}

func (h *FriendHandler) RemoveFriend(c *fiber.Ctx) error {
	userID, _ := middleware.UserIDFromContext(c)
	friendshipID := c.Params("id")

	if err := h.friendships.Remove(c.Context(), friendshipID, userID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fiber.ErrInternalServerError
	}

	return c.Redirect("/friends")
}
