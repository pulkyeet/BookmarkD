package database

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/pulkyeet/BookmarkD/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(email, username, passwordHash string) (*models.User, error) {
	query := `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, username, created_at, updated_at
	`
	user := &models.User{}
	err := r.db.QueryRow(query, email, username, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				if pqErr.Constraint == "users_email_key" {
					return nil, models.ErrEmailExists
				}
				if pqErr.Constraint == "users_username_key" {
					return nil, models.ErrUsernameExists
				}
			}

		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, username, password_hash, google_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	var passwordNull sql.NullString
	var googleIDNull sql.NullString
	
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&passwordNull,
		&googleIDNull,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, models.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	
	user.PasswordHash = passwordNull.String
	if googleIDNull.Valid {
		user.GoogleID = &googleIDNull.String
	}
	
	return user, nil
}

func (r *UserRepository) GetByID(userID int) (*models.User, error) {
	query := `SELECT id, email, username, created_at, updated_at
FROM users
WHERE id = $1`

	user := &models.User{}
	err := r.db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (r *UserRepository) GetProfile(userID int, viewerID *int) (*models.UserProfile, error) {
	query := `
    SELECT
        u.id, u.email, u.username, u.created_at, u.updated_at,
        COUNT(DISTINCT r.id) as total_books,
        COALESCE(AVG(r.rating), 0) as avg_rating,
        COUNT(DISTINCT CASE WHEN r.status = 'to_read' THEN r.id END) as to_read,
        COUNT(DISTINCT CASE WHEN r.status = 'currently_reading' THEN r.id END) as currently_reading,
        COUNT(DISTINCT CASE WHEN r.status = 'finished_reading' THEN r.id END) as finished,
        (SELECT COUNT(*) FROM follows WHERE following_id = u.id) as followers_count,
        (SELECT COUNT(*) FROM follows WHERE follower_id = u.id) as following_count
    FROM users u
    LEFT JOIN ratings r ON u.id = r.user_id
    WHERE u.id = $1
    GROUP BY u.id`

	profile := &models.UserProfile{}
	err := r.db.QueryRow(query, userID).Scan(
		&profile.ID,
		&profile.Email,
		&profile.Username,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.TotalBooks,
		&profile.AverageRating,
		&profile.ToReadCount,
		&profile.CurrentlyReadingCount,
		&profile.FinishedReadingCount,
		&profile.FollowersCount,
		&profile.FollowingCount,
	)
	if err != nil {
		return nil, err
	}

	// Check if viewer is following this user
	if viewerID != nil && *viewerID != userID {
		followQuery := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)`
		r.db.QueryRow(followQuery, *viewerID, userID).Scan(&profile.IsFollowing)
	}

	return profile, nil
}

func (r *UserRepository) GetByGoogleID(googleID string) (*models.User, error) {
	query := `
		SELECT id, email, username, password_hash, google_id, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`
	user := &models.User{}
	var passwordNull sql.NullString
	var googleIDStr sql.NullString
	
	err := r.db.QueryRow(query, googleID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&passwordNull,
		&googleIDStr,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	
	user.PasswordHash = passwordNull.String
	if googleIDStr.Valid {
		user.GoogleID = &googleIDStr.String
	}
	
	return user, nil
}

func (r *UserRepository) CreateWithGoogle(email, username, googleID string) (*models.User, error) {
	query := `INSERT INTO users (email, username, google_id) VALUES ($1, $2, $3) RETURNING id, email, username, google_id, created_at, updated_at`
	user := &models.User{}
	err := r.db.QueryRow(query, email, username, googleID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				if pqErr.Constraint == "users_email_key" {
					return nil, models.ErrEmailExists
				}
				if pqErr.Constraint == "users_username_key" {
					return nil, models.ErrUsernameExists
				}
			}
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) LinkGoogleAccount(userID int, googleID string) error {
	query := `UPDATE users SET google_id = $1 WHERE id = $2`
	_, err := r.db.Exec(query, googleID, userID)
	return err
}
