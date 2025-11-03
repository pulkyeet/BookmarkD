package models

import (
	"time"
)

type List struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Public      bool      `json:"public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListWithBooks struct {
	List
	Username string     `json:"username"`
	Books    []ListBook `json:"books"`
}

type ListBook struct {
	BookID   int       `json:"book_id"`
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	CoverURL string    `json:"cover_url"`
	Position int       `json:"position"`
	AddedAt  time.Time `json:"added_at"`
}
