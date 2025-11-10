package handlers

import (
	"encoding/json"
	"github.com/pulkyeet/BookmarkD/internal/database"
	"log"
	"net/http"
)

type GenreHandler struct {
	genreRepo *database.GenreRepository
}

func NewGenreHandler(genreRepo *database.GenreRepository) *GenreHandler {
	return &GenreHandler{genreRepo: genreRepo}
}

func (h *GenreHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	genres, err := h.genreRepo.GetAll()
	if err != nil {
		log.Println("Error fetching genres:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(genres)
}
