package database

import (
	"database/sql"
	"fmt"
	"github.com/pulkyeet/bookrate/internal/models"
)

type RatingRepository struct {
	db *sql.DB
}

func NewRatingRepository(db *sql.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

// Create or update rating
func (r *RatingRepository) Upsert(userID, bookID, rating int, review string, status string) (*models.Rating, error) {
	query := `
INSERT INTO ratings (user_id, book_id, rating, review, status)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, book_id)
DO UPDATE SET
	rating = EXCLUDED.rating,
	review = EXCLUDED.review,
	status = EXCLUDED.status,
	updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, book_id, rating, review, status, created_at, updated_at`

	ratingModel := &models.Rating{}
	var reviewNull sql.NullString

	err := r.db.QueryRow(query, userID, bookID, rating, nullString(review), status).Scan(
		&ratingModel.ID,
		&ratingModel.UserID,
		&ratingModel.BookID,
		&ratingModel.Rating,
		&reviewNull,
		&ratingModel.Status,
		&ratingModel.CreatedAt,
		&ratingModel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ratingModel.Review = reviewNull.String
	return ratingModel, nil
}

// Get all ratings for a book with stats
func (r *RatingRepository) GetByBookID(bookID int) (*models.BookRatingStats, error) {
	statsQuery := `
SELECT 
	COALESCE(AVG(rating), 0) AS average_rating,
	COUNT(*) as total
FROM ratings
WHERE book_id = $1`

	stats := &models.BookRatingStats{BookID: bookID}
	err := r.db.QueryRow(statsQuery, bookID).Scan(&stats.AverageRating, &stats.TotalRatings)
	if err != nil {
		return nil, err
	}

	ratingsQuery := `
SELECT
	r.id, r.user_id, r.book_id, r.rating, r.review, r.created_at, r.updated_at,
	u.username
FROM ratings r
JOIN users u ON r.user_id = u.id
WHERE r.book_id = $1
ORDER BY r.created_at DESC`

	rows, err := r.db.Query(ratingsQuery, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.RatingWithUser{}
	for rows.Next() {
		var rating models.RatingWithUser
		var reviewNull sql.NullString
		var ratingValue int64
		err := rows.Scan(
			&rating.ID,
			&rating.UserID,
			&rating.BookID,
			&ratingValue,
			&reviewNull,
			&rating.CreatedAt,
			&rating.UpdatedAt,
			&rating.Username,
		)
		if err != nil {
			return nil, err
		}
		rating.Rating.Rating = int(ratingValue)
		rating.Review = reviewNull.String
		ratings = append(ratings, rating)
	}
	stats.Ratings = ratings
	return stats, nil
}

func (r *RatingRepository) GetByUserAndBook(userID, bookID int) (*models.Rating, error) {
	query := `
SELECT id, user_id, book_id, rating, review, status, created_at, updated_at
FROM ratings
WHERE user_id = $1 AND book_id = $2`
	rating := &models.Rating{}
	var reviewNull sql.NullString

	err := r.db.QueryRow(query, userID, bookID).Scan(
		&rating.ID,
		&rating.UserID,
		&rating.BookID,
		&rating.Rating,
		&reviewNull,
		&rating.Status,
		&rating.CreatedAt,
		&rating.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rating.Review = reviewNull.String
	return rating, nil
}

func (r *RatingRepository) GetByUserID(userID int) ([]models.Rating, error) {
	query := `
SELECT id, user_id, book_id, rating, review, created_at, updated_at
FROM ratings
WHERE user_id = $1
ORDER BY created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.Rating{}
	for rows.Next() {
		var rating models.Rating
		var reviewNull sql.NullString

		err := rows.Scan(
			&rating.ID,
			&rating.UserID,
			&rating.BookID,
			&rating.Rating,
			&reviewNull,
			&rating.CreatedAt,
			&rating.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rating.Review = reviewNull.String
		ratings = append(ratings, rating)
	}
	return ratings, nil
}

func (r *RatingRepository) Delete(userID, bookID int) error {
	query := `
DELETE FROM ratings
WHERE user_id = $1 AND book_id = $2`
	result, err := r.db.Exec(query, userID, bookID)
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

func (r *RatingRepository) GetFeed(userID *int, limit, offset int) ([]models.FeedItem, error) {
	query := `
    SELECT 
        r.id, r.user_id, r.book_id, r.rating, r.review, r.status, r.created_at, r.updated_at,
        u.username,
        b.title, b.author, b.cover_url
    FROM ratings r
    JOIN users u ON r.user_id = u.id
    JOIN books b ON r.book_id = b.id`

	args := []interface{}{}
	argCount := 1

	if userID != nil {
		query += ` WHERE r.user_id IN (SELECT following_id FROM follows WHERE follower_id = $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, *userID)
		argCount++
	}

	query += ` ORDER BY r.created_at DESC LIMIT $` + fmt.Sprintf("%d", argCount) + ` OFFSET $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.FeedItem{}
	for rows.Next() {
		var item models.FeedItem
		var reviewNull sql.NullString
		var coverNull sql.NullString
		var ratingValue int

		err := rows.Scan(
			&item.ID, &item.UserID, &item.BookID, &ratingValue, &reviewNull, &item.Status,
			&item.CreatedAt, &item.UpdatedAt,
			&item.Username,
			&item.BookTitle, &item.BookAuthor, &coverNull,
		)
		if err != nil {
			return nil, err
		}
		item.Rating.Rating = ratingValue
		item.Review = reviewNull.String
		item.BookCover = coverNull.String
		items = append(items, item)
	}
	return items, nil
}

func (r *RatingRepository) GetByUserIDWithStatus(userID int, status string) ([]models.Rating, error) {
	query := `SELECT id, user_id, book_id, rating, review, status, created_at, updated_at
FROM ratings
WHERE user_id = $1`

	args := []interface{}{userID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.Rating{}
	for rows.Next() {
		var rating models.Rating
		var reviewNull sql.NullString

		err := rows.Scan(&rating.ID, &rating.UserID, &rating.BookID, &rating.Rating, &reviewNull, &rating.Status, &rating.CreatedAt, &rating.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rating.Review = reviewNull.String
		ratings = append(ratings, rating)
	}
	return ratings, nil
}
