package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FriendshipRepository struct {
	db *pgxpool.Pool
}

func NewFriendshipRepository(db *pgxpool.Pool) *FriendshipRepository {
	return &FriendshipRepository{db: db}
}

// GetBetween returns the friendship row between two users, in either direction, if any exists.
func (r *FriendshipRepository) GetBetween(ctx context.Context, userA, userB string) (*models.Friendship, error) {
	const query = `
		SELECT id, requester_id, addressee_id, status, created_at
		FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)`

	row := r.db.QueryRow(ctx, query, userA, userB)
	f, err := scanFriendship(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

func (r *FriendshipRepository) Create(ctx context.Context, requesterID, addresseeID string) (*models.Friendship, error) {
	const query = `
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, requester_id, addressee_id, status, created_at`

	row := r.db.QueryRow(ctx, query, requesterID, addresseeID)
	return scanFriendship(row)
}

// Accept marks a pending request as accepted. Only the addressee may accept it.
func (r *FriendshipRepository) Accept(ctx context.Context, friendshipID, addresseeID string) error {
	const query = `
		UPDATE friendships SET status = 'accepted'
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'`

	tag, err := r.db.Exec(ctx, query, friendshipID, addresseeID)
	if err != nil {
		return fmt.Errorf("accepting friendship: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Reject deletes a pending request. Only the addressee may reject it.
func (r *FriendshipRepository) Reject(ctx context.Context, friendshipID, addresseeID string) error {
	const query = `DELETE FROM friendships WHERE id = $1 AND addressee_id = $2 AND status = 'pending'`

	tag, err := r.db.Exec(ctx, query, friendshipID, addresseeID)
	if err != nil {
		return fmt.Errorf("rejecting friendship: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Remove deletes an accepted friendship. Either party may remove it.
func (r *FriendshipRepository) Remove(ctx context.Context, friendshipID, userID string) error {
	const query = `
		DELETE FROM friendships
		WHERE id = $1 AND status = 'accepted' AND (requester_id = $2 OR addressee_id = $2)`

	tag, err := r.db.Exec(ctx, query, friendshipID, userID)
	if err != nil {
		return fmt.Errorf("removing friendship: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListFriends returns the accepted friends of a user, paired with the friendship ID.
func (r *FriendshipRepository) ListFriends(ctx context.Context, userID string) ([]models.Friend, error) {
	const query = `
		SELECT f.id, u.id, u.username, u.email, u.password_hash, u.avatar_url, u.bio, u.is_premium, u.created_at
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
		ORDER BY u.username`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing friends: %w", err)
	}
	defer rows.Close()

	var friends []models.Friend
	for rows.Next() {
		var f models.Friend
		err := rows.Scan(&f.FriendshipID, &f.User.ID, &f.User.Username, &f.User.Email,
			&f.User.PasswordHash, &f.User.AvatarURL, &f.User.Bio, &f.User.IsPremium, &f.User.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning friend: %w", err)
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

// ListFriendIDs returns just the user IDs of a user's accepted friends.
func (r *FriendshipRepository) ListFriendIDs(ctx context.Context, userID string) ([]string, error) {
	const query = `
		SELECT CASE WHEN requester_id = $1 THEN addressee_id ELSE requester_id END
		FROM friendships
		WHERE status = 'accepted' AND (requester_id = $1 OR addressee_id = $1)`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing friend ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListPendingIncoming returns pending requests sent to a user, along with the requester's details.
func (r *FriendshipRepository) ListPendingIncoming(ctx context.Context, userID string) ([]models.FriendRequest, error) {
	const query = `
		SELECT f.id, u.id, u.username, u.email, u.password_hash, u.avatar_url, u.bio, u.is_premium, u.created_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing pending requests: %w", err)
	}
	defer rows.Close()

	var requests []models.FriendRequest
	for rows.Next() {
		var fr models.FriendRequest
		err := rows.Scan(&fr.FriendshipID, &fr.From.ID, &fr.From.Username, &fr.From.Email,
			&fr.From.PasswordHash, &fr.From.AvatarURL, &fr.From.Bio, &fr.From.IsPremium, &fr.From.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning pending request: %w", err)
		}
		requests = append(requests, fr)
	}
	return requests, rows.Err()
}

// CountPendingIncoming returns how many pending requests a user has received.
func (r *FriendshipRepository) CountPendingIncoming(ctx context.Context, userID string) (int, error) {
	const query = `SELECT count(*) FROM friendships WHERE addressee_id = $1 AND status = 'pending'`

	var count int
	if err := r.db.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting pending requests: %w", err)
	}
	return count, nil
}

func scanFriendship(row rowScanner) (*models.Friendship, error) {
	var f models.Friendship
	err := row.Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
