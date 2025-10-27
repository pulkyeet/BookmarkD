package middleware

import (
	"context"
	"github.com/pulkyeet/bookrate/internal/models"
	"log"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}
		token := parts[1]
		claims, err := models.ValidateToken(token)
		if err != nil {
			log.Printf("Token validation error: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetUserFromContext(r *http.Request) (*models.Claims, bool) {
	claims, ok := r.Context().Value(UserContextKey).(*models.Claims)
	return claims, ok
}
