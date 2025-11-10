package handlers

import (
	"encoding/json"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/middleware"
	"log"
	"net/http"
	"strconv"
)

type FeedHandler struct {
	ratingRepo *database.RatingRepository
}

func NewFeedHandler(ratingRepo *database.RatingRepository) *FeedHandler {
	return &FeedHandler{ratingRepo: ratingRepo}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	var userID *int
	if claims, ok := middleware.GetUserFromContext(r); ok {
		userID = &claims.UserID
	}

	feedType := r.URL.Query().Get("type")
	if feedType == "" {
		feedType = "all"
	}

	limit := 20
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	items, err := h.ratingRepo.GetFeedByType(userID, feedType, limit, offset)
	if err != nil {
		log.Printf("Error getting feed: %v", err)
		http.Error(w, "Failed to get feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}