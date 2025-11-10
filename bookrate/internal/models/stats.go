package models

type UserYearStats struct {
	Year             int                `json:"year"`
	BooksRead        int                `json:"books_read"`
	AverageRating    float64            `json:"average_rating"`
	TopGenres        []GenreCount       `json:"top_genres"`
	FavouriteAuthors []AuthorCount      `json:"favourite_authors"`
	ReadingStreak    int                `json:"reading_streak"`
	MonthlyActivity  []MonthlyBookCount `json:"monthly_activity"`
}

type GenreCount struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

type AuthorCount struct {
	Author string `json:"author"`
	Count  int    `json:"count"`
}

type MonthlyBookCount struct {
	Month int `json:"month"`
	Count int `json:"count"`
}
