package database

import (
	"database/sql"
	"github.com/pulkyeet/bookrate/internal/models"
)

type GenreRepository struct {
	db *sql.DB
}

func NewGenreRepository(db *sql.DB) *GenreRepository {
	return &GenreRepository{db: db}
}

func (r *GenreRepository) GetAll() ([]models.Genre, error) {
	query := `SELECT id, name, created_at FROM genres ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	genres := []models.Genre{}
	for rows.Next() {
		var genre models.Genre
		err := rows.Scan(&genre.ID, &genre.Name, &genre.CreatedAt)
		if err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}
	return genres, nil
}

func (r *GenreRepository) GetByBookID(bookID int) ([]models.Genre, error) {
	query := `SELECT g.id, g.name, g.created_at FROM genres g JOIN book_genres b ON g.id = b.genre_id WHERE b.book_id = $1 ORDER BY g.name ASC`
	rows, err := r.db.Query(query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	genres := []models.Genre{}
	for rows.Next() {
		var genre models.Genre
		err := rows.Scan(&genre.ID, &genre.Name, &genre.CreatedAt)
		if err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}
	return genres, nil
}

func (r *GenreRepository) AddGenreToBook(bookID, genreID int) error {
	query := `INSERT INTO book_genres (book_id, genre_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(query, bookID, genreID)
	return err
}

func (r *GenreRepository) GetByName(name string) (*models.Genre, error) {
	query := `SELECT id, name, created_at FROM genres WHERE LOWER(name) = LOWER($1)`
	genre := &models.Genre{}
	err := r.db.QueryRow(query, name).Scan(&genre.ID, &genre.Name, &genre.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return genre, nil
}
