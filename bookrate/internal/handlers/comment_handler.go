package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/middleware"
	"log"
	"net/http"
	"strconv"
)

type CommentHandler struct {
	commentRepo *database.CommentRepository
}

func NewCommentHandler(commentRepo *database.CommentRepository) *CommentHandler {
	return &CommentHandler{commentRepo: commentRepo}
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	ratingID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid Rating ID", http.StatusBadRequest)
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "Comment Text is required", http.StatusBadRequest)
		return
	}
	comment, err := h.commentRepo.Create(claims.UserID, ratingID, req.Text)
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func (h *CommentHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	ratingID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid Rating ID", http.StatusBadRequest)
		return
	}
	comments, err := h.commentRepo.GetByRatingID(ratingID)
	if err != nil {
		log.Printf("Error getting comments: %v", err)
		http.Error(w, "Failed to get comments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	commentID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid Comment ID", http.StatusBadRequest)
		return
	}
	err = h.commentRepo.Delete(commentID, claims.UserID)
	if err == sql.ErrNoRows {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error deleting comment: %v", err)
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
