package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	GoogleID     *string   `json:"google_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserProfile struct {
	User
	TotalBooks            int     `json:"total_books"`
	AverageRating         float64 `json:"average_rating"`
	ToReadCount           int     `json:"to_read_count"`
	CurrentlyReadingCount int     `json:"currently_reading"`
	FinishedReadingCount  int     `json:"finished_reading"`
	FollowersCount        int     `json:"followers_count"`
	FollowingCount        int     `json:"following_count"`
	IsFollowing           bool    `json:"is_following"`
}
