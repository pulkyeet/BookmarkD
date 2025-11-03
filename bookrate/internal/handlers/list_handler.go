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

type ListHandler struct {
	listRepo *database.ListRepository
}

func NewListHandler(listRepo *database.ListRepository) *ListHandler {
	return &ListHandler{listRepo: listRepo}
}

func (h *ListHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Public      bool   `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Error decoding request body:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}
	list, err := h.listRepo.Create(claims.UserID, req.Name, req.Description, req.Public)
	if err != nil {
		log.Println("Error creating list:", err)
		http.Error(w, "Failed to create list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(list)
}

func (h *ListHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	list, err := h.listRepo.GetByID(listID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Error getting list:", err)
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *ListHandler) GetUserLists(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println("Error decoding userID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	lists, err := h.listRepo.GetByUserID(userID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Error getting lists:", err)
		http.Error(w, "Failed to get lists", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

func (h *ListHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Public      bool   `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Error decoding request body:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}
	list, err := h.listRepo.Update(listID, claims.UserID, req.Name, req.Description, req.Public)
	if err != nil {
		log.Println("Error updating list:", err)
		http.Error(w, "Failed to update list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *ListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	err = h.listRepo.Delete(listID, claims.UserID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Error deleting list:", err)
		http.Error(w, "Failed to delete list", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) AddBook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Printf("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	var req struct {
		BookID   int `json:"book_id"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Printf("DEBUG Handler received: BookID=%d, Position=%d", req.BookID, req.Position)
	list, err := h.listRepo.GetByID(listID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error getting list:", err)
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}
	if list.UserID != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	position := req.Position
	if position == 0 {
		position, err = h.listRepo.GetNextPosition(listID)
		if err != nil {
			log.Printf("Error getting next position:", err)
			http.Error(w, "Failed to add book", http.StatusInternalServerError)
			return
		}

	}
	err = h.listRepo.AddBook(listID, req.BookID, position)
	if err != nil {
		log.Printf("Error adding book:", err)
		http.Error(w, "Failed to add book", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) RemoveBook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Printf("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	bookID, err := strconv.Atoi(r.PathValue("bookID"))
	if err != nil {
		log.Printf("Error decoding bookID:", err)
		http.Error(w, "Invalid Book ID", http.StatusBadRequest)
		return
	}
	list, err := h.listRepo.GetByID(listID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error getting list:", err)
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}
	if list.UserID != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	err = h.listRepo.RemoveBook(listID, bookID)
	if err == sql.ErrNoRows {
		http.Error(w, "Book not found in list", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error removing book from list", err)
		http.Error(w, "Failed to remove book from list", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) ReorderBooks(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Printf("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Books []struct {
			BookID   int `json:"book_id"`
			Position int `json:"position"`
		} `json:"books"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	list, err := h.listRepo.GetByID(listID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error getting list:", err)
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}
	if list.UserID != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	bookPositions := make(map[int]int)
	for _, book := range req.Books {
		bookPositions[book.BookID] = book.Position
	}
	err = h.listRepo.ReorderBooks(listID, bookPositions)
	if err != nil {
		log.Printf("Error reordering books from list", err)
		http.Error(w, "Failed to reorder books from list", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) BookmarkList(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Printf("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	err = h.listRepo.BookmarkList(claims.UserID, listID)
	if err != nil {
		log.Printf("Error bookmarking list:", err)
		http.Error(w, "Failed to bookmark list", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) UnbookmarkList(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	listID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Printf("Error decoding listID:", err)
		http.Error(w, "Invalid List ID", http.StatusBadRequest)
		return
	}
	err = h.listRepo.UnbookmarkList(claims.UserID, listID)
	if err == sql.ErrNoRows {
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error unbookmarking list:", err)
		http.Error(w, "Failed to unbookmark list", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListHandler) GetBookmarkedLists(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	lists, err := h.listRepo.GetBookmarkedLists(claims.UserID)
	if err != nil {
		log.Printf("Error getting bookmarked lists:", err)
		http.Error(w, "Failed to get lists", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

func (h *ListHandler) GetPopularLists(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	lists, err := h.listRepo.GetPopularLists(limit)
	if err != nil {
		log.Printf("Error getting popular lists:", err)
		http.Error(w, "Failed to get lists", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}
