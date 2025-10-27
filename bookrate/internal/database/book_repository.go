package database

import (
	"database/sql"
	"fmt"
	"github.com/pulkyeet/bookrate/internal/models"
	"strings"
)

type BookRepository struct {
	db *sql.DB
}

func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

func (r *BookRepository) Create(req models.CreateBookRequest) (*models.Book, error) {
	query := `
		INSERT INTO books (title, author, isbn, description, published_year, cover_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, author, isbn, description, published_year, cover_url, created_at, updated_at
		`

	book := &models.Book{}
	var isbn, description, coverURL sql.NullString
	var publishedYear sql.NullInt64
	err := r.db.QueryRow(
		query,
		req.Title,
		req.Author,
		nullString(req.ISBN),
		nullString(req.Description),
		nullInt(req.PublishedYear),
		nullString(req.CoverURL),
	).Scan(
		&book.ID,
		&book.Title,
		&book.Author,
		&isbn,
		&description,
		&publishedYear,
		&coverURL,
		&book.CreatedAt,
		&book.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	book.ISBN = isbn.String
	book.Description = description.String
	book.PublishedYear = int(publishedYear.Int64)
	book.CoverURL = coverURL.String
	return book, nil
}

func (r *BookRepository) GetByID(id int) (*models.Book, error) {
	query := `
		SELECT id, title, author, isbn, description, published_year, cover_url, created_at, updated_at
		FROM books
		WHERE id = $1`

	book := &models.Book{}
	var isbn, description, coverURL sql.NullString
	var publishedYear sql.NullInt64
	err := r.db.QueryRow(query, id).Scan(
		&book.ID,
		&book.Title,
		&book.Author,
		&isbn,
		&description,
		&publishedYear,
		&coverURL,
		&book.CreatedAt,
		&book.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("book not found")
	}
	if err != nil {
		return nil, err
	}
	book.ISBN = isbn.String
	book.Description = description.String
	book.PublishedYear = int(publishedYear.Int64)
	book.CoverURL = coverURL.String
	return book, nil
}

func (r *BookRepository) List(limit, offset int, search string) ([]*models.Book, error) {
	query := `
		SELECT id, title, author, isbn, description, published_year, cover_url, created_at, updated_at
		FROM books
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR author ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []*models.Book{}
	for rows.Next() {
		book := &models.Book{}
		var isbn, description, coverURL sql.NullString
		var publishedYear sql.NullInt64

		err := rows.Scan(
			&book.ID,
			&book.Title,
			&book.Author,
			&isbn,
			&description,
			&publishedYear,
			&coverURL,
			&book.CreatedAt,
			&book.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert nullable types
		book.ISBN = isbn.String
		book.Description = description.String
		book.PublishedYear = int(publishedYear.Int64)
		book.CoverURL = coverURL.String

		books = append(books, book)
	}

	return books, nil
}

func (r *BookRepository) Update(id int, req models.UpdateBookRequest) (*models.Book, error) {
	// Building dynamic update query
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argCount))
		args = append(args, *req.Title)
		argCount++
	}
	if req.Author != nil {
		updates = append(updates, fmt.Sprintf("author = $%d", argCount))
		args = append(args, *req.Author)
		argCount++
	}
	if req.ISBN != nil {
		updates = append(updates, fmt.Sprintf("isbn = $%d", argCount))
		args = append(args, *req.ISBN)
		argCount++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *req.Description)
		argCount++
	}
	if req.PublishedYear != nil {
		updates = append(updates, fmt.Sprintf("published_year = $%d", argCount))
		args = append(args, *req.PublishedYear)
		argCount++
	}
	if req.CoverURL != nil {
		updates = append(updates, fmt.Sprintf("cover_url = $%d", argCount))
		args = append(args, *req.CoverURL)
		argCount++
	}
	if len(updates) == 0 {
		return r.GetByID(id)
	}
	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE books
		SET %s
		WHERE id = $%d
		RETURNING id, title, author, isbn, description, published_year, cover_url, created_at, updated_at
		`, strings.Join(updates, ", "), argCount)

	book := &models.Book{}
	err := r.db.QueryRow(query, args...).Scan(
		&book.ID,
		&book.Title,
		&book.Author,
		&book.ISBN,
		&book.Description,
		&book.PublishedYear,
		&book.CoverURL,
		&book.CreatedAt,
		&book.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Book not found.")
	}
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (r *BookRepository) Delete(id int) error {
	query := `DELETE FROM books WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("Book now found.")
	}
	return nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
