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
func (r *MoodRepository) Upsert(ctx context.Context, userID string, score float64, emoji string, note *string, entryDate time.Time) (*models.MoodEntry, error) {
	const query = `
		INSERT INTO mood_entries (user_id, mood_score, mood_emoji, note, entry_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, entry_date)
		DO UPDATE SET mood_score = EXCLUDED.mood_score, mood_emoji = EXCLUDED.mood_emoji, note = EXCLUDED.note
		RETURNING id, user_id, mood_score, mood_emoji, note, entry_date, created_at`

	row := r.db.QueryRow(ctx, query, userID, score, emoji, note, entryDate)
	return scanMoodEntry(row)
}

// Delete removes a user's mood entry for the given date, if any.
func (r *MoodRepository) Delete(ctx context.Context, userID string, entryDate time.Time) error {
	const query = `DELETE FROM mood_entries WHERE user_id = $1 AND entry_date = $2`

	if _, err := r.db.Exec(ctx, query, userID, entryDate); err != nil {
		return fmt.Errorf("deleting mood entry: %w", err)
	}
	return nil
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

// ListByUserSince returns a user's mood entries from the given date onward, oldest first.
func (r *MoodRepository) ListByUserSince(ctx context.Context, userID string, since time.Time) ([]models.MoodEntry, error) {
	const query = `
		SELECT id, user_id, mood_score, mood_emoji, note, entry_date, created_at
		FROM mood_entries
		WHERE user_id = $1 AND entry_date >= $2
		ORDER BY entry_date ASC`

	rows, err := r.db.Query(ctx, query, userID, since)
	if err != nil {
		return nil, fmt.Errorf("listing mood entries since: %w", err)
	}
	defer rows.Close()

	return collectMoodEntries(rows)
}

// ListEntryDates returns all dates a user has logged a mood, oldest first.
func (r *MoodRepository) ListEntryDates(ctx context.Context, userID string) ([]time.Time, error) {
	const query = `SELECT entry_date FROM mood_entries WHERE user_id = $1 ORDER BY entry_date ASC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing entry dates: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// LastActivityAt returns the timestamp of a user's most recently created mood
// entry. The second return value is false if the user has no entries yet.
func (r *MoodRepository) LastActivityAt(ctx context.Context, userID string) (time.Time, bool, error) {
	const query = `SELECT max(created_at) FROM mood_entries WHERE user_id = $1`

	var lastActivity *time.Time
	if err := r.db.QueryRow(ctx, query, userID).Scan(&lastActivity); err != nil {
		return time.Time{}, false, fmt.Errorf("getting last activity: %w", err)
	}
	if lastActivity == nil {
		return time.Time{}, false, nil
	}
	return *lastActivity, true, nil
}

// Stats returns the all-time average mood score and total entry count for a user.
func (r *MoodRepository) Stats(ctx context.Context, userID string) (average float64, count int, err error) {
	const query = `SELECT coalesce(avg(mood_score), 0), count(*) FROM mood_entries WHERE user_id = $1`

	if err := r.db.QueryRow(ctx, query, userID).Scan(&average, &count); err != nil {
		return 0, 0, fmt.Errorf("computing mood stats: %w", err)
	}
	return average, count, nil
}

// StatsBetween returns the average mood score and entry count for a user within
// an inclusive date range. Used for weekly/monthly/yearly breakdowns.
func (r *MoodRepository) StatsBetween(ctx context.Context, userID string, start, end time.Time) (average float64, count int, err error) {
	const query = `
		SELECT coalesce(avg(mood_score), 0), count(*)
		FROM mood_entries
		WHERE user_id = $1 AND entry_date BETWEEN $2 AND $3`

	if err := r.db.QueryRow(ctx, query, userID, start, end).Scan(&average, &count); err != nil {
		return 0, 0, fmt.Errorf("computing mood stats between: %w", err)
	}
	return average, count, nil
}

// ListFeedForUserIDs returns recent mood entries for the given users, each paired
// with the posting user's details, newest first.
func (r *MoodRepository) ListFeedForUserIDs(ctx context.Context, userIDs []string, limit int) ([]models.FeedEntry, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	const query = `
		SELECT u.id, u.username, u.email, u.password_hash, u.avatar_url, u.bio, u.is_premium, u.created_at,
		       m.id, m.user_id, m.mood_score, m.mood_emoji, m.note, m.entry_date, m.created_at
		FROM mood_entries m
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = ANY($1)
		ORDER BY m.entry_date DESC, m.created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("listing feed entries: %w", err)
	}
	defer rows.Close()

	var entries []models.FeedEntry
	for rows.Next() {
		var fe models.FeedEntry
		err := rows.Scan(
			&fe.User.ID, &fe.User.Username, &fe.User.Email, &fe.User.PasswordHash,
			&fe.User.AvatarURL, &fe.User.Bio, &fe.User.IsPremium, &fe.User.CreatedAt,
			&fe.Mood.ID, &fe.Mood.UserID, &fe.Mood.Score, &fe.Mood.Emoji, &fe.Mood.Note,
			&fe.Mood.EntryDate, &fe.Mood.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning feed entry: %w", err)
		}
		entries = append(entries, fe)
	}
	return entries, rows.Err()
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
