package handlers

import (
	"encoding/json"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type FeedHandler struct {
	ratingRepo *database.RatingRepository
}

func NewFeedHandler(ratingRepo *database.RatingRepository) *FeedHandler {
	return &FeedHandler{ratingRepo: ratingRepo}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	feedType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var userID *int
	if feedType == "following" {
		// Parse token manually for optional auth
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized - login to see following feed", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := models.ValidateToken(parts[1])
		if err != nil {
			log.Printf("Token validation error: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userID = &claims.UserID
	}

	feed, err := h.ratingRepo.GetFeed(userID, limit, offset)
	if err != nil {
		log.Printf("GetFeed error: %v", err)
		http.Error(w, "Failed to get feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}
