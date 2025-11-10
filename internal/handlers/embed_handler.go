package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/pulkyeet/BookmarkD/internal/database"
	"log"
	"net/http"
	"strconv"
)

type EmbedHandler struct {
	ratingRepo *database.RatingRepository
	listRepo   *database.ListRepository
	userRepo   *database.UserRepository
}

func NewEmbedHandler(ratingRepo *database.RatingRepository, listRepo *database.ListRepository, userRepo *database.UserRepository) *EmbedHandler {
	return &EmbedHandler{ratingRepo: ratingRepo, listRepo: listRepo, userRepo: userRepo}
}

func (h *EmbedHandler) GetUserBooks(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		return
	}
	count := 5
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 20 {
			count = c
		}
	}
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}
	books, err := h.ratingRepo.GetTopRatedByUser(userID, count)
	if err != nil {
		log.Printf("Error getting books for user %v: %v", userID, err)
		http.Error(w, "Failed to get books", http.StatusInternalServerError)
		return
	}
	response := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"books":    books,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *EmbedHandler) GetListBooks(w http.ResponseWriter, r *http.Request) {
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid list id", http.StatusBadRequest)
		return
	}
	count := 5
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 20 {
			count = c
		}
	}
	list, err := h.listRepo.GetByID(listID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error getting list %v: %v", listID, err)
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}
	if !list.Public {
		http.Error(w, "List is not public", http.StatusForbidden)
		return
	}
	books := list.Books
	if len(books) > count {
		books = books[:count]
	}
	response := map[string]interface{}{
		"list_id":     list.ID,
		"list_name":   list.Name,
		"description": list.Description,
		"username":    list.Username,
		"user_id":     list.UserID,
		"books":       books,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
