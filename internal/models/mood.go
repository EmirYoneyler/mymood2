package models

import (
	"fmt"
	"math"
	"time"
)

type MoodEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Score     float64   `json:"mood_score"`
	Emoji     string    `json:"mood_emoji"`
	Note      *string   `json:"note,omitempty"`
	EntryDate time.Time `json:"entry_date"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteText returns the entry's note, or an empty string if none was set.
func (e MoodEntry) NoteText() string {
	if e.Note == nil {
		return ""
	}
	return *e.Note
}

// ScoreText formats the score with a single decimal place, e.g. "7.6".
func (e MoodEntry) ScoreText() string {
	return fmt.Sprintf("%.1f", e.Score)
}

// MoodEmoji returns the emoji associated with a 1.0-10.0 mood score,
// rounded to the nearest whole point.
func MoodEmoji(score float64) string {
	emojis := map[int]string{
		1:  "😭",
		2:  "😢",
		3:  "😞",
		4:  "😕",
		5:  "😐",
		6:  "🙂",
		7:  "😊",
		8:  "😄",
		9:  "😁",
		10: "🤩",
	}

	rounded := int(math.Round(score))
	if rounded < 1 {
		rounded = 1
	}
	if rounded > 10 {
		rounded = 10
	}

	return emojis[rounded]
}
