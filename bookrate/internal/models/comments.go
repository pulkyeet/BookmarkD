package models

import "time"

type Comment struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RatingID  int       `json:"rating_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentWithUser struct {
	Comment
	Username string `json:"username"`
}
