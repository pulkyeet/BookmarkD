package database

import (
	"database/sql"
	"github.com/pulkyeet/bookrate/internal/models"
)

type FollowRepository struct {
	db *sql.DB
}

func NewFollowRepository(db *sql.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Follow(followerID, followingID int) error {
	query := `INSERT INTO follows (follower_id, following_id) values ($1, $2)`
	_, err := r.db.Exec(query, followerID, followingID)
	return err
}

func (r *FollowRepository) Unfollow(followerID, followingID int) error {
	query := `DELETE FROM follows where follower_id = $1 AND following_id = $2`
	result, err := r.db.Exec(query, followerID, followingID)

	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FollowRepository) IsFollowing(followerID, followingID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM followes WHERE follower_id = $1 AND following_id = $2)`
	var exists bool
	err := r.db.QueryRow(query, followerID, followingID).Scan(&exists)
	return exists, err
}

func (r *FollowRepository) GetFollowers(userID int) ([]models.User, error) {
	query := `SELECT u.id, u.email, u.username, u.created_at, u.updated_at
	FROM users u
	JOIN follows f ON u.id = f.follower_id
	WHERE f.following_id = $1
	ORDER BY f.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *FollowRepository) GetFollowing(userID int) ([]models.User, error) {
	query := `SELECT u.id, u.email, u.username, u.created_at, u.updated_at
	FROM users u
	JOIN follows f ON u.id = f.following_id
	WHERE f.follower_id = $1
	ORDER BY f.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *FollowRepository) GetFollowCounts(userID int) (int, int, error) {
	query := `
    SELECT 
        (SELECT COUNT(*) FROM follows WHERE following_id = $1) as followers,
        (SELECT COUNT(*) FROM follows WHERE follower_id = $1) as following`

	var followers, following int
	err := r.db.QueryRow(query, userID).Scan(&followers, &following)
	return followers, following, err
}
