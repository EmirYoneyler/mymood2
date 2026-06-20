package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emiryoneyler/mymood/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MoodRepository struct {
	db *pgxpool.Pool
}

func NewMoodRepository(db *pgxpool.Pool) *MoodRepository {
	return &MoodRepository{db: db}
}

// Upsert creates or updates the mood entry for the given user and date.
func (r *MoodRepository) Upsert(ctx context.Context, userID string, score int, emoji string, note *string, entryDate time.Time) (*models.MoodEntry, error) {
	const query = `
		INSERT INTO mood_entries (user_id, mood_score, mood_emoji, note, entry_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, entry_date)
		DO UPDATE SET mood_score = EXCLUDED.mood_score, mood_emoji = EXCLUDED.mood_emoji, note = EXCLUDED.note
		RETURNING id, user_id, mood_score, mood_emoji, note, entry_date, created_at`

	row := r.db.QueryRow(ctx, query, userID, score, emoji, note, entryDate)
	return scanMoodEntry(row)
}

func (r *MoodRepository) GetByUserAndDate(ctx context.Context, userID string, entryDate time.Time) (*models.MoodEntry, error) {
	const query = `
		SELECT id, user_id, mood_score, mood_emoji, note, entry_date, created_at
		FROM mood_entries WHERE user_id = $1 AND entry_date = $2`

	row := r.db.QueryRow(ctx, query, userID, entryDate)
	entry, err := scanMoodEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return entry, err
}

// ListByUser returns the most recent mood entries for a user, newest first.
func (r *MoodRepository) ListByUser(ctx context.Context, userID string, limit int) ([]models.MoodEntry, error) {
	const query = `
		SELECT id, user_id, mood_score, mood_emoji, note, entry_date, created_at
		FROM mood_entries
		WHERE user_id = $1
		ORDER BY entry_date DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing mood entries: %w", err)
	}
	defer rows.Close()

	return collectMoodEntries(rows)
}

// ListByUserIDs returns recent mood entries across multiple users (e.g. a friend feed), newest first.
func (r *MoodRepository) ListByUserIDs(ctx context.Context, userIDs []string, limit int) ([]models.MoodEntry, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	const query = `
		SELECT id, user_id, mood_score, mood_emoji, note, entry_date, created_at
		FROM mood_entries
		WHERE user_id = ANY($1)
		ORDER BY entry_date DESC, created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("listing mood entries: %w", err)
	}
	defer rows.Close()

	return collectMoodEntries(rows)
}

func collectMoodEntries(rows pgx.Rows) ([]models.MoodEntry, error) {
	var entries []models.MoodEntry
	for rows.Next() {
		entry, err := scanMoodEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func scanMoodEntry(row rowScanner) (*models.MoodEntry, error) {
	var e models.MoodEntry
	err := row.Scan(&e.ID, &e.UserID, &e.Score, &e.Emoji, &e.Note, &e.EntryDate, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning mood entry: %w", err)
	}
	return &e, nil
}
