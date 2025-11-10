package database

import (
	"database/sql"
	"github.com/pulkyeet/bookrate/internal/models"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(userID, ratingID int, text string) (*models.Comment, error) {
	query := `INSERT INTO comments (user_id, rating_id, text) VALUES ($1, $2, $3) RETURNING id, user_id, rating_id, text, created_at`

	comment := &models.Comment{}
	err := r.db.QueryRow(query, userID, ratingID, text).Scan(&comment.ID, &comment.UserID, &comment.RatingID, &comment.Text, &comment.CreatedAt)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *CommentRepository) GetByRatingID(ratingID int) ([]models.CommentWithUser, error) {
	query := `SELECT c.id, c.user_id, c.rating_id, c.text, c.created_at, u.username
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.rating_id = $1`

	rows, err := r.db.Query(query, ratingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []models.CommentWithUser{}
	for rows.Next() {
		var comment models.CommentWithUser
		err := rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.RatingID,
			&comment.Text,
			&comment.CreatedAt,
			&comment.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *CommentRepository) Delete(commentID, userID int) error {
	query := `DELETE FROM comments where id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, commentID, userID)
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
