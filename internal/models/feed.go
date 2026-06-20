package models

// FeedEntry pairs a mood entry with the user who posted it, for display in a feed.
type FeedEntry struct {
	User User
	Mood MoodEntry
}
