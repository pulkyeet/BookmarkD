package database

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
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

func (r *BookRepository) ListWithGenres(limit, offset int, search, genreFilter string) ([]*models.BookWithGenres, error) {
	query := `SELECT DISTINCT b.id, b.title, b.author, b.isbn, b.description, b.published_year, b.cover_url, b.created_at, b.updated_at
FROM books b LEFT JOIN book_genres bg ON b.id = bg.book_id LEFT JOIN genres g ON bg.genre_id = g.id WHERE 1=1`
	args := []interface{}{}
	argCount := 1
	if search != "" {
		query += fmt.Sprintf(" AND (b.title ILIKE $%d OR b.author ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}
	if genreFilter != "" {
		query += fmt.Sprintf(" AND LOWER(g.name) = LOWER($%d)", argCount)
		args = append(args, genreFilter)
		argCount++
	}
	query += " ORDER BY b.created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookMap := make(map[int]*models.BookWithGenres)
	bookOrder := []int{}

	for rows.Next() {
		var bookID int
		book := &models.Book{}
		var isbn, description, coverURL sql.NullString
		var publishedYear sql.NullInt64

		err := rows.Scan(
			&bookID,
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

		if _, exists := bookMap[bookID]; !exists {
			book.ID = bookID
			book.ISBN = isbn.String
			book.Description = description.String
			book.PublishedYear = int(publishedYear.Int64)
			book.CoverURL = coverURL.String

			bookMap[bookID] = &models.BookWithGenres{
				Book:   *book,
				Genres: []models.Genre{},
			}
			bookOrder = append(bookOrder, bookID)
		}
	}

	// Now fetch genres for all books
	bookIDs := make([]int, 0, len(bookMap))
	for id := range bookMap {
		bookIDs = append(bookIDs, id)
	}

	if len(bookIDs) > 0 {
		genreQuery := `
			SELECT bg.book_id, g.id, g.name, g.created_at
			FROM book_genres bg
			JOIN genres g ON bg.genre_id = g.id
			WHERE bg.book_id = ANY($1)
			ORDER BY g.name ASC`

		genreRows, err := r.db.Query(genreQuery, pq.Array(bookIDs))
		if err != nil {
			return nil, err
		}
		defer genreRows.Close()

		for genreRows.Next() {
			var bookID int
			var genre models.Genre
			err := genreRows.Scan(&bookID, &genre.ID, &genre.Name, &genre.CreatedAt)
			if err != nil {
				return nil, err
			}
			if book, exists := bookMap[bookID]; exists {
				book.Genres = append(book.Genres, genre)
			}
		}
	}

	// Return in original order
	books := make([]*models.BookWithGenres, 0, len(bookOrder))
	for _, id := range bookOrder {
		books = append(books, bookMap[id])
	}

	return books, nil
}

func (r *BookRepository) GetByIDWithGenres(id int) (*models.BookWithGenres, error) {
	book, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	genreQuery := `
		SELECT g.id, g.name, g.created_at
		FROM genres g
		JOIN book_genres bg ON g.id = bg.genre_id
		WHERE bg.book_id = $1
		ORDER BY g.name ASC`

	rows, err := r.db.Query(genreQuery, id)
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

	return &models.BookWithGenres{
		Book:   *book,
		Genres: genres,
	}, nil
}

// GetSimilarBooks - Collaborative filtering
func (r *BookRepository) GetSimilarBooks(bookID, limit int) ([]map[string]interface{}, error) {
	query := `
		WITH users_who_liked AS (
			SELECT DISTINCT user_id
			FROM ratings
			WHERE book_id = $1 AND rating >= 7
		),
		other_books_liked AS (
			SELECT r.book_id, COUNT(DISTINCT r.user_id) as common_users
			FROM ratings r
			INNER JOIN users_who_liked u ON r.user_id = u.user_id
			WHERE r.book_id != $1 AND r.rating >= 7
			GROUP BY r.book_id
			HAVING COUNT(DISTINCT r.user_id) >= 2
		)
		SELECT 
			b.id, b.title, b.author, b.cover_url,
			obl.common_users,
			COALESCE(AVG(r.rating), 0) as avg_rating
		FROM other_books_liked obl
		JOIN books b ON obl.book_id = b.id
		LEFT JOIN ratings r ON b.id = r.book_id
		GROUP BY b.id, b.title, b.author, b.cover_url, obl.common_users
		ORDER BY obl.common_users DESC, avg_rating DESC
		LIMIT $2`

	rows, err := r.db.Query(query, bookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []map[string]interface{}{}
	for rows.Next() {
		var bookID, commonUsers int
		var title, author string
		var coverURL sql.NullString
		var avgRating float64

		err := rows.Scan(&bookID, &title, &author, &coverURL, &commonUsers, &avgRating)
		if err != nil {
			return nil, err
		}

		books = append(books, map[string]interface{}{
			"book_id":      bookID,
			"title":        title,
			"author":       author,
			"cover_url":    coverURL.String,
			"common_users": commonUsers,
			"avg_rating":   avgRating,
		})
	}
	return books, nil
}

// GetTrendingBooks - Most rated in the last X days
func (r *BookRepository) GetTrendingBooks(days, limit int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT 
			b.id, b.title, b.author, b.cover_url,
			COUNT(r.id) as rating_count,
			COALESCE(AVG(r.rating), 0) as avg_rating
		FROM ratings r
		JOIN books b ON r.book_id = b.id
		WHERE r.created_at >= NOW() - INTERVAL '%d days'
		GROUP BY b.id, b.title, b.author, b.cover_url
		HAVING COUNT(r.id) >= 1
		ORDER BY rating_count DESC, avg_rating DESC
		LIMIT $1`, days)

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []map[string]interface{}{}
	for rows.Next() {
		var bookID, ratingCount int
		var title, author string
		var coverURL sql.NullString
		var avgRating float64

		err := rows.Scan(&bookID, &title, &author, &coverURL, &ratingCount, &avgRating)
		if err != nil {
			return nil, err
		}

		books = append(books, map[string]interface{}{
			"book_id":      bookID,
			"title":        title,
			"author":       author,
			"cover_url":    coverURL.String,
			"rating_count": ratingCount,
			"avg_rating":   avgRating,
		})
	}
	return books, nil
}

// GetPopularBooks - Highest avg rating (min X ratings)
func (r *BookRepository) GetPopularBooks(minRatings, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			b.id, b.title, b.author, b.cover_url,
			COALESCE(AVG(r.rating), 0) as avg_rating,
			COUNT(r.id) as rating_count
		FROM books b
		JOIN ratings r ON b.id = r.book_id
		GROUP BY b.id, b.title, b.author, b.cover_url
		HAVING COUNT(r.id) >= $1
		ORDER BY avg_rating DESC, rating_count DESC
		LIMIT $2`

	rows, err := r.db.Query(query, minRatings, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []map[string]interface{}{}
	for rows.Next() {
		var bookID, ratingCount int
		var title, author string
		var coverURL sql.NullString
		var avgRating float64

		err := rows.Scan(&bookID, &title, &author, &coverURL, &avgRating, &ratingCount)
		if err != nil {
			return nil, err
		}

		books = append(books, map[string]interface{}{
			"book_id":      bookID,
			"title":        title,
			"author":       author,
			"cover_url":    coverURL.String,
			"avg_rating":   avgRating,
			"rating_count": ratingCount,
		})
	}
	return books, nil
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

func (r *BookRepository) FindByTitleAuthor(title, author string) (*models.Book, error) {
	query := `SELECT id, title, author, isbn, description, published_year, cover_url, created_at, updated_at FROM books WHERE LOWER(title) = LOWER($1) AND LOWER(author) = LOWER($2) LIMIT 1`
	book := &models.Book{}
	var isbnNull, descNull, coverNull sql.NullString
	var yearNull sql.NullInt64
	
	err := r.db.QueryRow(query, title, author).Scan(&book.ID, &book.Title, &book.Author, &isbnNull, &descNull, &yearNull, &coverNull, &book.CreatedAt, &book.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	book.CoverURL = coverNull.String
	book.Description = descNull.String
	book.ISBN = isbnNull.String
	if yearNull.Valid {
		book.PublishedYear = int(yearNull.Int64)
	}
	return book, nil
}