package database

import (
	"database/sql"
	"github.com/pulkyeet/BookmarkD/internal/models"
	"log"
)

type ListRepository struct {
	db *sql.DB
}

func NewListRepository(db *sql.DB) *ListRepository {
	return &ListRepository{db: db}
}

func (r *ListRepository) Create(userID int, name, description string, public bool) (*models.List, error) {
	query := `INSERT INTO lists (user_id, name, description, public) VALUES ($1, $2, $3, $4) RETURNING id, user_id, name, description, public, created_at, updated_at`
	list := &models.List{}
	err := r.db.QueryRow(query, userID, name, nullString(description), public).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Description, &list.Public, &list.CreatedAt, &list.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ListRepository) GetByID(listID int) (*models.ListWithBooks, error) {
	listQuery := `SELECT l.id, l.user_id, l.name, l.description, l.public, l.created_at, l.updated_at, u.username
FROM lists l
JOIN users u ON l.user_id = u.id
WHERE l.id = $1`
	list := &models.ListWithBooks{}
	var descNull sql.NullString
	err := r.db.QueryRow(listQuery, listID).Scan(&list.ID, &list.UserID, &list.Name, &descNull, &list.Public, &list.CreatedAt, &list.UpdatedAt, &list.Username)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	list.Description = descNull.String
	booksQuery := `SELECT lb.book_id, b.title, b.author, b.cover_url, lb.position, lb.added_at
FROM list_books lb
JOIN books b on lb.book_id = b.id
WHERE lb.list_id = $1
ORDER BY lb.position ASC`
	rows, err := r.db.Query(booksQuery, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []models.ListBook{}
	for rows.Next() {
		var book models.ListBook
		var coverNull sql.NullString
		err := rows.Scan(&book.BookID, &book.Title, &book.Author, &coverNull, &book.Position, &book.AddedAt)
		if err != nil {
			return nil, err
		}
		book.CoverURL = coverNull.String
		books = append(books, book)
	}
	list.Books = books
	return list, nil
}

func (r *ListRepository) GetByUserID(userID int) ([]models.List, error) {
	query := `SELECT id, user_id, name, description, public, created_at, updated_at 
			  FROM lists WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := []models.List{}
	for rows.Next() {
		var list models.List
		var descNull sql.NullString
		err := rows.Scan(&list.ID, &list.UserID, &list.Name, &descNull, &list.Public, &list.CreatedAt, &list.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list.Description = descNull.String
		lists = append(lists, list)
	}
	return lists, nil
}

func (r *ListRepository) Update(listID, userID int, name, description string, public bool) (*models.List, error) {
	query := `UPDATE lists SET name = $1, description = $2, public = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 AND user_id = $5 RETURNING id, user_id, name, description, public, created_at, updated_at`
	list := &models.List{}
	var descNull sql.NullString
	err := r.db.QueryRow(query, name, nullString(description), public, listID, userID).Scan(&list.ID, &list.UserID, &list.Name, &descNull, &list.Public, &list.CreatedAt, &list.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	list.Description = descNull.String
	return list, nil
}
func (r *ListRepository) Delete(listID, userID int) error {
	query := `DELETE FROM lists WHERE id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, listID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ListRepository) AddBook(listID, bookID, position int) error {
	log.Printf("DEBUG AddBook: listID=%d, bookID=%d, position=%d", listID, bookID, position)
	query := `INSERT INTO list_books (list_id, book_id, position) VALUES ($1, $2, $3) ON CONFLICT (list_id, book_id) DO NOTHING`
	_, err := r.db.Exec(query, listID, bookID, position)
	if err != nil {
		log.Printf("DEBUG AddBook error: %v", err)
	}
	return err
}

func (r *ListRepository) RemoveBook(listID, bookID int) error {
	query := `DELETE FROM list_books WHERE list_id = $1 AND book_id = $2`
	result, err := r.db.Exec(query, listID, bookID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *ListRepository) ReorderBooks(listID int, bookPositions map[int]int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`UPDATE list_books SET position = $1 WHERE list_id = $2 AND book_id = $3`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for bookID, position := range bookPositions {
		_, err := stmt.Exec(position, listID, bookID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ListRepository) GetNextPosition(listID int) (int, error) {
	query := `SELECT COALESCE(MAX(position), 0) + 1 FROM list_books WHERE list_id = $1`
	var position int
	err := r.db.QueryRow(query, listID).Scan(&position)
	return position, err
}

func (r *ListRepository) BookmarkList(userID, listID int) error {
	query := `INSERT INTO list_bookmarks (user_id, list_id) VALUES ($1, $2) ON CONFLICT (user_id, list_id) DO NOTHING`
	_, err := r.db.Exec(query, userID, listID)
	return err
}

func (r *ListRepository) UnbookmarkList(userID, listID int) error {
	query := `DELETE FROM list_bookmarks WHERE user_id = $1 AND list_id = $2`
	result, err := r.db.Exec(query, userID, listID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ListRepository) GetBookmarkedLists(userID int) ([]models.List, error) {
	query := `SELECT l.id, l.user_id, l.name, l.description, l.public, l.created_at, l.updated_at
FROM lists l
JOIN list_bookmarks lb on l.id = lb.list_id
WHERE lb.user_id = $1
ORDER BY lb.created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lists := []models.List{}
	for rows.Next() {
		var list models.List
		var descNull sql.NullString
		err := rows.Scan(&list.ID, &list.UserID, &list.Name, &descNull, &list.Public, &list.CreatedAt, &list.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list.Description = descNull.String
		lists = append(lists, list)
	}
	return lists, nil
}

func (r *ListRepository) GetPopularLists(limit int) ([]models.List, error) {
	query := `SELECT l.id, l.user_id, l.name, l.description, l.public, l.created_at, l.updated_at, COUNT(lb.user_id) AS bookmark_count
FROM lists l
LEFT JOIN list_bookmarks lb on l.id = lb.list_id
WHERE l.public = true
GROUP BY l.id, l.user_id, l.name, l.description, l.public, l.created_at, l.updated_at
ORDER BY bookmark_count DESC, l.created_at DESC LIMIT $1`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lists := []models.List{}
	for rows.Next() {
		var list models.List
		var descNull sql.NullString
		var bookmarkCount int
		err := rows.Scan(&list.ID, &list.UserID, &list.Name, &descNull, &list.Public, &list.CreatedAt, &list.UpdatedAt, &bookmarkCount)
		if err != nil {
			return nil, err
		}
		list.Description = descNull.String
		lists = append(lists, list)
	}
	return lists, nil
}
