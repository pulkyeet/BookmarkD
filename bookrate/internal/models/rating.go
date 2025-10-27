package models

import "time"

type Rating struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	BookID    int       `json:"book_id"`
	Rating    int       `json:"rating"`
	Review    string    `json:"review,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RatingWithUser struct {
	Rating
	Username string `json:"username"`
}

type BookRatingStats struct {
	BookID        int              `json:"book_id"`
	AverageRating float64          `json:"average_rating"`
	TotalRatings  int              `json:"total_ratings"`
	Ratings       []RatingWithUser `json:"ratings"`
}

type FeedItem struct {
	Rating
	Username   string `json:"username"`
	BookTitle  string `json:"book_title"`
	BookAuthor string `json:"book_author"`
	BookCover  string `json:"book_cover"`
}
