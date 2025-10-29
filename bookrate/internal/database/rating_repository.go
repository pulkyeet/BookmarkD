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
func (r *RatingRepository) GetByBookID(bookID int, userID *int, sortBy string) (*models.BookRatingStats, error) {
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
	u.username,
	COUNT(DISTINCT rl.user_id) as like_count,
	COUNT(DISTINCT c.id) as comment_count`

	if userID != nil {
		ratingsQuery += `,
	EXISTS(SELECT 1 FROM review_likes WHERE user_id = $2 AND rating_id = r.id) as liked_by_user`
	}

	ratingsQuery += `
FROM ratings r
JOIN users u ON r.user_id = u.id
LEFT JOIN review_likes rl ON r.id = rl.rating_id
LEFT JOIN comments c ON r.id = c.rating_id
WHERE r.book_id = $1
GROUP BY r.id, r.user_id, r.book_id, r.rating, r.review, r.created_at, r.updated_at, u.username`

	switch sortBy {
	case "most_liked":
		ratingsQuery += ` ORDER BY like_count DESC, r.created_at DESC`
	case "highest_rating":
		ratingsQuery += ` ORDER BY r.rating DESC, r.created_at DESC`
	case "newest":
		fallthrough
	default:
		ratingsQuery += ` ORDER BY r.created_at DESC`
	}

	var rows *sql.Rows
	if userID != nil {
		rows, err = r.db.Query(ratingsQuery, bookID, *userID)
	} else {
		rows, err = r.db.Query(ratingsQuery, bookID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.RatingWithLikes{}
	for rows.Next() {
		var rating models.RatingWithLikes
		var reviewNull sql.NullString
		var ratingValue int64

		if userID != nil {
			err := rows.Scan(
				&rating.ID, &rating.UserID, &rating.BookID, &ratingValue, &reviewNull,
				&rating.CreatedAt, &rating.UpdatedAt, &rating.Username,
				&rating.LikeCount, &rating.CommentCount, &rating.LikedByUser,
			)
			if err != nil {
				return nil, err
			}
		} else {
			err := rows.Scan(
				&rating.ID, &rating.UserID, &rating.BookID, &ratingValue, &reviewNull,
				&rating.CreatedAt, &rating.UpdatedAt, &rating.Username,
				&rating.LikeCount, &rating.CommentCount,
			)
			if err != nil {
				return nil, err
			}
			rating.LikedByUser = false
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

func (r *RatingRepository) GetFeed(requestingUserID *int, limit, offset int) ([]models.FeedItem, error) {
	query := `
    SELECT 
        r.id, r.user_id, r.book_id, r.rating, r.review, r.status, r.created_at, r.updated_at,
        u.username,
        b.title, b.author, b.cover_url,
        COUNT(DISTINCT rl.user_id) as like_count,
        COUNT(DISTINCT c.id) as comment_count`

	if requestingUserID != nil {
		query += `,
        EXISTS(SELECT 1 FROM review_likes WHERE user_id = $1 AND rating_id = r.id) as liked_by_user`
	}

	query += `
    FROM ratings r
    JOIN users u ON r.user_id = u.id
    JOIN books b ON r.book_id = b.id
    LEFT JOIN review_likes rl ON r.id = rl.rating_id
    LEFT JOIN comments c ON r.id = c.rating_id`

	args := []interface{}{}
	argCount := 1

	if requestingUserID != nil {
		args = append(args, *requestingUserID)
		argCount++
		query += ` WHERE r.user_id IN (SELECT following_id FROM follows WHERE follower_id = $1)`
	}

	query += ` GROUP BY r.id, r.user_id, r.book_id, r.rating, r.review, r.status, r.created_at, r.updated_at, u.username, b.title, b.author, b.cover_url`
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

		if requestingUserID != nil {
			err := rows.Scan(
				&item.ID, &item.UserID, &item.BookID, &ratingValue, &reviewNull, &item.Status,
				&item.CreatedAt, &item.UpdatedAt, &item.Username,
				&item.BookTitle, &item.BookAuthor, &coverNull,
				&item.LikeCount, &item.CommentCount, &item.LikedByUser,
			)
			if err != nil {
				return nil, err
			}
		} else {
			err := rows.Scan(
				&item.ID, &item.UserID, &item.BookID, &ratingValue, &reviewNull, &item.Status,
				&item.CreatedAt, &item.UpdatedAt, &item.Username,
				&item.BookTitle, &item.BookAuthor, &coverNull,
				&item.LikeCount, &item.CommentCount,
			)
			if err != nil {
				return nil, err
			}
			item.LikedByUser = false
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

func (r *RatingRepository) LikeRating(userID, ratingID int) error {
	query := `INSERT INTO review_likes VALUES ($1, $2) ON CONFLICT (user_id, rating_id) DO NOTHING`

	_, err := r.db.Exec(query, userID, ratingID)
	return err
}

func (r *RatingRepository) UnlikeRating(userID, ratingID int) error {
	query := `DELETE FROM review_likes WHERE user_id = $1 AND rating_id = $2`
	result, err := r.db.Exec(query, userID, ratingID)
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

func (r *RatingRepository) GetLikeCount(ratingID int) (int, error) {
	query := `SELECT COUNT(*) FROM review_likes where rating_id = $1`

	var count int
	err := r.db.QueryRow(query, ratingID).Scan(&count)

	return count, err
}

func (r *RatingRepository) HasUserLiked(userID, ratingID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM review_likes WHERE user_id = $1 AND rating_id = $2)`

	var exists bool
	err := r.db.QueryRow(query, userID, ratingID).Scan(&exists)
	return exists, err
}

func (r *RatingRepository) Update(ratingID, userID, rating int, review string) (*models.Rating, error) {
	query := `UPDATE ratings
SET rating = $1, review = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $3 AND user_id = $4
RETURNING id, user_id, book_id, rating, review, status, created_at, updated_at`

	ratingModel := &models.Rating{}
	var reviewNull sql.NullString

	err := r.db.QueryRow(query, rating, nullString(review), ratingID, userID).Scan(
		&ratingModel.ID, &ratingModel.UserID, &ratingModel.BookID, &ratingModel.Rating, &reviewNull, &ratingModel.Status, &ratingModel.CreatedAt, &ratingModel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	ratingModel.Review = reviewNull.String
	return ratingModel, nil
}
