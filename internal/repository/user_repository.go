package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	const query = `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, avatar_url, bio, is_premium, created_at`

	return r.scanOne(ctx, query, username, email, passwordHash)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const query = `
		SELECT id, username, email, password_hash, avatar_url, bio, is_premium, created_at
		FROM users WHERE email = $1`

	return r.scanOne(ctx, query, email)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	const query = `
		SELECT id, username, email, password_hash, avatar_url, bio, is_premium, created_at
		FROM users WHERE username = $1`

	return r.scanOne(ctx, query, username)
}

// Delete permanently removes a user and, via ON DELETE CASCADE, their mood
// entries and friendships.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM users WHERE id = $1`

	if _, err := r.db.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	const query = `
		SELECT id, username, email, password_hash, avatar_url, bio, is_premium, created_at
		FROM users WHERE id = $1`

	return r.scanOne(ctx, query, id)
}

// SearchByUsername returns users whose username matches the given prefix, excluding excludeUserID.
func (r *UserRepository) SearchByUsername(ctx context.Context, prefix, excludeUserID string) ([]models.User, error) {
	const query = `
		SELECT id, username, email, password_hash, avatar_url, bio, is_premium, created_at
		FROM users
		WHERE username ILIKE $1 || '%' AND id != $2
		ORDER BY username
		LIMIT 20`

	rows, err := r.db.Query(ctx, query, prefix, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *UserRepository) scanOne(ctx context.Context, query string, args ...any) (*models.User, error) {
	row := r.db.QueryRow(ctx, query, args...)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	return u, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.AvatarURL, &u.Bio, &u.IsPremium, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
