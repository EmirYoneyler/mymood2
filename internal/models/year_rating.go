package models

import (
	"fmt"
	"time"
)

// YearRating is a holistic, retrospective score a user gives to an entire
// year (e.g. "2024 was an 8/10 for me"), independent of daily mood entries.
type YearRating struct {
	ID        string
	UserID    string
	Year      int
	Score     float64
	Note      *string
	CreatedAt time.Time
}

func (r YearRating) NoteText() string {
	if r.Note == nil {
		return ""
	}
	return *r.Note
}

func (r YearRating) ScoreText() string {
	return fmt.Sprintf("%.1f", r.Score)
}
