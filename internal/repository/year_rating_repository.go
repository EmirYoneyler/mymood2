package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type YearRatingRepository struct {
	db *pgxpool.Pool
}

func NewYearRatingRepository(db *pgxpool.Pool) *YearRatingRepository {
	return &YearRatingRepository{db: db}
}

// Upsert creates or updates a user's rating for the given year.
func (r *YearRatingRepository) Upsert(ctx context.Context, userID string, year int, score float64, note *string) (*models.YearRating, error) {
	const query = `
		INSERT INTO year_ratings (user_id, year, score, note)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, year)
		DO UPDATE SET score = EXCLUDED.score, note = EXCLUDED.note
		RETURNING id, user_id, year, score, note, created_at`

	row := r.db.QueryRow(ctx, query, userID, year, score, note)
	return scanYearRating(row)
}

func (r *YearRatingRepository) GetByYear(ctx context.Context, userID string, year int) (*models.YearRating, error) {
	const query = `
		SELECT id, user_id, year, score, note, created_at
		FROM year_ratings WHERE user_id = $1 AND year = $2`

	row := r.db.QueryRow(ctx, query, userID, year)
	rating, err := scanYearRating(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rating, err
}

// Delete removes a user's rating for the given year, if any.
func (r *YearRatingRepository) Delete(ctx context.Context, userID string, year int) error {
	const query = `DELETE FROM year_ratings WHERE user_id = $1 AND year = $2`

	if _, err := r.db.Exec(ctx, query, userID, year); err != nil {
		return fmt.Errorf("deleting year rating: %w", err)
	}
	return nil
}

func scanYearRating(row rowScanner) (*models.YearRating, error) {
	var yr models.YearRating
	err := row.Scan(&yr.ID, &yr.UserID, &yr.Year, &yr.Score, &yr.Note, &yr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning year rating: %w", err)
	}
	return &yr, nil
}
