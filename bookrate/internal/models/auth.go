package models

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type PendingOAuthClaims struct {
	Email    string `json:"email"`
	GoogleID string `json:"google_id"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists.")
	ErrUsernameExists     = errors.New("username already exists")
)

func GeneratePendingOAuthToken(email, googleID, name string) (string, error) {
	claims := PendingOAuthClaims{
		Email:    email,
		GoogleID: googleID,
		Name:     name,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func ValidatePendingOAuthToken(tokenString string) (*PendingOAuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PendingOAuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*PendingOAuthClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("Invalid token")
}
