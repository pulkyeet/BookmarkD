package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/middleware"
	"github.com/pulkyeet/bookrate/internal/models"
)

type ImportHandler struct {
	bookRepo *database.BookRepository
	ratingRepo *database.RatingRepository
}

type ImportResult struct {
	BookImported int `json:"books_imported"`
	RatingsImported int `json:"ratings_imported"`
	Skipped int `json:"skipped"`
	Errors []string `json:"errors"`
}

func NewImportHandler(bookRepo *database.BookRepository, ratingRepo *database.RatingRepository) *ImportHandler {
	return &ImportHandler{
		bookRepo: bookRepo,
		ratingRepo: ratingRepo,
	}
}

func (h *ImportHandler) ImportGoodreads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return
	}
	err := r.ParseMultipartForm(10 << 20) 
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("csv")
	if err != nil {
		http.Error(w, "No file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "failed to parse csv", http.StatusBadRequest)
		return
	}
	if len(records) < 2 {
		http.Error(w, "csv is empty", http.StatusBadRequest)
		return
	}
	header := records[0]
	titleIdx := findColumn(header, "Title")
	authorIdx := findColumn(header, "Author")
	isbnIdx := findColumn(header, "ISBN13")
	ratingIdx := findColumn(header, "My Rating")
	shelfIdx := findColumn(header, "Exclusive Shelf")
	
	if titleIdx == -1 || authorIdx == -1 {
		http.Error(w, "CSV missing required columns(title, author)", http.StatusBadRequest)
		return
	}
	result := ImportResult{
		Errors: []string{},
	}
	for i, record := range records[1:] {
		if len(record) <= max(titleIdx, authorIdx, ratingIdx, shelfIdx, isbnIdx) {
			result.Skipped++
			continue
		}
		title := strings.TrimSpace(record[titleIdx])
		author := strings.TrimSpace(record[authorIdx])
		if title == "" || author == "" {
			result.Skipped++
			continue
		}
		book, err := h.bookRepo.FindByTitleAuthor(title, author)
		if err != nil {
			isbn := ""
			if isbnIdx!= -1 && len(record) > isbnIdx {
				isbn = strings.TrimSpace(record[isbnIdx])
			}
			book, err = h.bookRepo.Create(models.CreateBookRequest{
				Title: title,
				Author: author,
				ISBN: isbn,
			})
			if err != nil {
				result.Errors = append(result.Errors, "Row "+strconv.Itoa(i+2)+": Failed to create book - "+title)
				continue
			}
			result.BookImported++
		}
		ratingVal := 0 
		if ratingIdx != -1 && len(record) > ratingIdx {
			ratingStr := strings.TrimSpace(record[ratingIdx])
			if ratingStr != "" && ratingStr != "0" {
				stars, err := strconv.Atoi(ratingStr)
				if err == nil && stars > 0 {
					ratingVal = stars * 2
				}
			}
		}
		status := "finished_reading"
		if shelfIdx != -1 && len(record) > shelfIdx {
			shelf := strings.ToLower(strings.TrimSpace(record[shelfIdx]))
			switch shelf {
				case "to-read": status = "to-read"
				case "currently-reading": status = "currently_reading"
				default: status = "finished_reading"
			}
		}
		if ratingVal > 0 {
			_, err = h.ratingRepo.Upsert(claims.UserID, book.ID, ratingVal, "", status)
			if err != nil {
				result.Errors = append(result.Errors, "Row "+strconv.Itoa(i+2)+": Failed to create rating - "+title)
				continue
			}
			result.RatingsImported++
		} else {
			result.Skipped++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func findColumn(header []string, name string) int {
	for i, col := range header {
		if strings.Contains(strings.ToLower(col), strings.ToLower(name)) {
			return i
		}
	}
	return -1
}

func max(nums ...int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n > m {
			m=n
		}
	}
	return m
}