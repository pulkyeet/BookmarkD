package database

import (
	"database/sql"
	"fmt"

	"github.com/pulkyeet/BookmarkD/internal/models"
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

func (r *RatingRepository) GetFeedByType(requestingUserID *int, feedType string, limit, offset int) ([]models.FeedItem, error) {
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

	// Apply following filter only if feedType is "following" AND user is logged in
	if feedType == "following" && requestingUserID != nil {
		query += ` WHERE r.user_id IN (SELECT following_id FROM follows WHERE follower_id = $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, *requestingUserID)
		argCount++
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

// Keep old GetFeed for backward compatibility if needed elsewhere
func (r *RatingRepository) GetFeed(requestingUserID *int, limit, offset int) ([]models.FeedItem, error) {
	return r.GetFeedByType(requestingUserID, "all", limit, offset)
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

func (r *RatingRepository) GetTopRatedByUser(userID, limit int) ([]map[string]interface{}, error) {
	query := `SELECT r.id as rating_id, r.rating, r.review, r.status, r.created_at, r.updated_at, b.id as book_id, b.title, b.author, b.cover_url FROM ratings r JOIN books b ON r.book_id = b.id WHERE r.user_id = $1 AND r.rating > 0 ORDER BY r.rating DESC, r.created_at DESC LIMIT $2`
	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	books := []map[string]interface{}{}
	for rows.Next() {
		var ratingID, bookID int
		var rating int
		var title, author string
		var review sql.NullString
		var status, createdAt, updatedAt, coverURL sql.NullString
		err := rows.Scan(&ratingID, &rating, &review, &status, &createdAt, &updatedAt, &bookID, &title, &author, &coverURL)
		if err != nil {
			return nil, err
		}
		reviewSnippet := ""
		if review.Valid && len(review.String) > 150 {
			reviewSnippet = review.String[:150] + "..."
		} else if review.Valid {
			reviewSnippet = review.String
		}
		books = append(books, map[string]interface{}{
			"rating_id":      ratingID,
			"book_id":        bookID,
			"title":          title,
			"author":         author,
			"cover_url":      coverURL.String,
			"rating":         rating,
			"review_snippet": reviewSnippet,
			"created_at":     createdAt.String,
			"updated_at":     updatedAt.String,
		})
	}
	return books, nil
}

func (r *RatingRepository) GetYearStats(userID, year int) (*models.UserYearStats, error) {
	stats := &models.UserYearStats{Year: year}
	countQuery := `SELECT COUNT(DISTINCT book_id) AS books_read, COALESCE(AVG(rating), 0) AS avg_rating FROM ratings WHERE user_id = $1 AND EXTRACT (YEAR FROM created_at) = $2 AND rating > 0`
	err := r.db.QueryRow(countQuery, userID, year).Scan(&stats.BooksRead, &stats.AverageRating)
	if err != nil {
		return nil, err
	}
	genresQuery := `SELECT g.name, COUNT(DISTINCT r.book_id) AS count FROM ratings r JOIN book_genres bg ON r.book_id = bg.book_id JOIN genres g ON bg.genre_id = g.id WHERE r.user_id = $1 AND EXTRACT(YEAR FROM r.created_at) = $2 AND r.rating > 0 GROUP BY g.name ORDER BY count DESC LIMIT 5`
	rows, err := r.db.Query(genresQuery, userID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.TopGenres = []models.GenreCount{}
	for rows.Next() {
		var gc models.GenreCount
		if err := rows.Scan(&gc.Genre, &gc.Count); err != nil {
			return nil, err
		}
		stats.TopGenres = append(stats.TopGenres, gc)
	}
	authorsQuery := `SELECT b.author, COUNT(DISTINCT r.book_id) AS count FROM ratings r JOIN books b ON r.book_id = b.id WHERE r.user_id = $1 AND EXTRACT(YEAR FROM r.created_at) = $2 AND r.rating > 0 GROUP BY b.author ORDER BY count DESC LIMIT 5`
	rows, err = r.db.Query(authorsQuery, userID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.FavouriteAuthors = []models.AuthorCount{}
	for rows.Next() {
		var ac models.AuthorCount
		if err := rows.Scan(&ac.Author, &ac.Count); err != nil {
			return nil, err
		}
		stats.FavouriteAuthors = append(stats.FavouriteAuthors, ac)
	}
	monthlyQuery := `SELECT EXTRACT(MONTH FROM created_at)::int AS month, COUNT (DISTINCT book_id) AS count FROM ratings WHERE user_id = $1 AND EXTRACT(YEAR FROM created_at) = $2 AND rating > 0 GROUP BY month ORDER BY month`
	rows, err = r.db.Query(monthlyQuery, userID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.MonthlyActivity = []models.MonthlyBookCount{}
	for rows.Next() {
		var mc models.MonthlyBookCount
		if err := rows.Scan(&mc.Month, &mc.Count); err != nil {
			return nil, err
		}
		stats.MonthlyActivity = append(stats.MonthlyActivity, mc)
	}
	streakQuery := `
			WITH daily_reads AS (
				SELECT DISTINCT DATE(created_at) as read_date
				FROM ratings
				WHERE user_id = $1
					AND EXTRACT(YEAR FROM created_at) = $2
					AND rating > 0
				ORDER BY read_date
			),
			streaks AS (
				SELECT
					read_date,
					read_date - (ROW_NUMBER() OVER (ORDER BY read_date))::int AS streak_group
				FROM daily_reads
			)
			SELECT COALESCE(MAX(streak_length), 0) as max_streak
			FROM (
				SELECT COUNT(*) as streak_length
				FROM streaks
				GROUP BY streak_group
			) sub
		`
	err = r.db.QueryRow(streakQuery, userID, year).Scan(&stats.ReadingStreak)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
