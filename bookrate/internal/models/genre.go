package models

import "time"

type Genre struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type BookWithGenres struct {
	Book
	Genres []Genre `json:"genres"`
}
