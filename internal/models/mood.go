package models

import "time"

type MoodEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Score     int       `json:"mood_score"`
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

// MoodEmoji returns the emoji associated with a 1-10 mood score.
func MoodEmoji(score int) string {
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
	if emoji, ok := emojis[score]; ok {
		return emoji
	}
	return "😐"
}
