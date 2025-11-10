package models

import "time"

type Book struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	ISBN          string    `json:"isbn,omitempty"`
	Description   string    `json:"description,omitempty"`
	PublishedYear int       `json:"published_year,omitempty"`
	CoverURL      string    `json:"cover_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateBookRequest struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn,omitempty"`
	Description   string `json:"description,omitempty"`
	PublishedYear int    `json:"published_year,omitempty"`
	CoverURL      string `json:"cover_url,omitempty"`
}

type UpdateBookRequest struct {
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	ISBN          *string `json:"isbn,omitempty"`
	Description   *string `json:"description,omitempty"`
	PublishedYear *int    `json:"published_year,omitempty"`
	CoverURL      *string `json:"cover_url,omitempty"`
}
