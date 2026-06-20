package models

import "time"

const (
	FriendshipPending  = "pending"
	FriendshipAccepted = "accepted"
)

type Friendship struct {
	ID          string    `json:"id"`
	RequesterID string    `json:"requester_id"`
	AddresseeID string    `json:"addressee_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// FriendRequest pairs a pending friendship with the user who sent it.
type FriendRequest struct {
	FriendshipID string
	From         User
}

// Friend pairs an accepted friendship with the other user in it.
type Friend struct {
	FriendshipID string
	User         User
}
