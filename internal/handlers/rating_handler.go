package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/pulkyeet/BookmarkD/internal/database"
	"github.com/pulkyeet/BookmarkD/internal/middleware"
	"log"
	"net/http"
	"strconv"
)

type RatingHandler struct {
	ratingRepo *database.RatingRepository
}

func NewRatingHandler(ratingRepo *database.RatingRepository) *RatingHandler {
	return &RatingHandler{ratingRepo: ratingRepo}
}

type CreateRatingRequest struct {
	Rating int    `json:"rating"`
	Review string `json:"review"`
}

func (h *RatingHandler) CreateRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	bookID, err := strconv.Atoi(r.URL.Query().Get("book_id"))
	if err != nil {
		http.Error(w, "Invalid book id", http.StatusBadRequest)
		return
	}

	var req struct {
		Rating int    `json:"rating"`
		Review string `json:"review"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default status to finished_reading if not provided
	if req.Status == "" {
		req.Status = "finished_reading"
	}

	// Validate status
	validStatuses := map[string]bool{
		"to_read":           true,
		"currently_reading": true,
		"finished_reading":  true,
	}
	if !validStatuses[req.Status] {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Rating validation: allow 0 for to_read/currently_reading, require 1-10 for finished
	if req.Status == "finished_reading" {
		if req.Rating < 1 || req.Rating > 10 {
			http.Error(w, "Rating must be between 1 and 10 for finished books", http.StatusBadRequest)
			return
		}
	} else {
		// For to_read and currently_reading, use 0 if not provided
		if req.Rating < 0 || req.Rating > 10 {
			http.Error(w, "Invalid rating", http.StatusBadRequest)
			return
		}
		// Force rating to 0 for non-finished books
		if req.Rating == 0 {
			req.Rating = 0 // Explicitly allow 0
		}
	}

	rating, err := h.ratingRepo.Upsert(userID, bookID, req.Rating, req.Review, req.Status)
	if err != nil {
		log.Printf("Upsert error: %v", err)
		http.Error(w, "Failed to create rating", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rating)
}

func (h *RatingHandler) GetBookRatings(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.Atoi(r.URL.Query().Get("book_id"))
	if err != nil {
		http.Error(w, "Invalid book id", http.StatusBadRequest)
		return
	}

	var userID *int
	if claims, ok := middleware.GetUserFromContext(r); ok {
		userID = &claims.UserID
	}

	sortBy := r.URL.Query().Get("sort_by")
	validSorts := map[string]bool{
		"newest":         true,
		"most_liked":     true,
		"highest_rating": true,
	}
	if sortBy != "" && !validSorts[sortBy] {
		http.Error(w, "Invalid sort_by parameter", http.StatusBadRequest)
		return
	}

	stats, err := h.ratingRepo.GetByBookID(bookID, userID, sortBy)
	if err != nil {
		log.Printf("Error getting ratings for book %d: %v", bookID, err)
		http.Error(w, "Failed to get ratings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *RatingHandler) GetMyRatings(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	status := r.URL.Query().Get("status")

	ratings, err := h.ratingRepo.GetByUserIDWithStatus(userID, status)
	if err != nil {
		http.Error(w, "Failed to get ratings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ratings)
}

func (h *RatingHandler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	bookID, err := strconv.Atoi(r.URL.Query().Get("book_id"))
	if err != nil {
		http.Error(w, "Invalid book id", http.StatusBadRequest)
		return
	}
	err = h.ratingRepo.Delete(userID, bookID)
	if err == sql.ErrNoRows {
		http.Error(w, "Rating not found", http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, "Failed to delete rating", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RatingHandler) GetMyRatingForBook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	bookID, err := strconv.Atoi(r.URL.Query().Get("book_id"))
	if err != nil {
		http.Error(w, "Invalid book id", http.StatusBadRequest)
		return
	}

	rating, err := h.ratingRepo.GetByUserAndBook(userID, bookID)
	if err != nil {
		http.Error(w, "Failed to get rating", http.StatusInternalServerError)
		return
	}

	if rating == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rating)
}

func (h *RatingHandler) LikeRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}
	ratingID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid ratinf ID", http.StatusBadRequest)
		return
	}
	err = h.ratingRepo.LikeRating(claims.UserID, ratingID)
	if err != nil {
		log.Printf("Error liking rating: %v", err)
		http.Error(w, "Failed to like rating", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RatingHandler) UnlikeRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	ratingID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid rating ID", http.StatusBadRequest)
		return
	}
	err = h.ratingRepo.UnlikeRating(claims.UserID, ratingID)
	if err == sql.ErrNoRows {
		http.Error(w, "Like not found", http.StatusInternalServerError)
		return
	}
	if err != nil {
		log.Printf("Error unliking rating: %v", err)
		http.Error(w, "Failed to unlike", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RatingHandler) UpdateRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	ratingID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid rating ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Rating int    `json:"rating"`
		Review string `json:"review"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}
	if req.Rating > 10 && req.Rating < 0 {
		http.Error(w, "Rating must be between 1 and 10", http.StatusBadRequest)
		return
	}
	rating, err := h.ratingRepo.Update(ratingID, claims.UserID, req.Rating, req.Review)
	if err == sql.ErrNoRows {
		http.Error(w, "Rating not found", http.StatusInternalServerError)
		return
	}
	if err != nil {
		log.Printf("Error updating rating: %v", err)
		http.Error(w, "Failed to update rating", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rating)
}
