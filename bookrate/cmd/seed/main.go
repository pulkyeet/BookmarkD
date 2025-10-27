package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/models"
)

type GoogleBooksResponse struct {
	Items []BookItem `json:"items"`
}

type BookItem struct {
	VolumeInfo VolumeInfo `json:"volumeInfo"`
}

type VolumeInfo struct {
	Title               string               `json:"title"`
	Authors             []string             `json:"authors"`
	PublishedDate       string               `json:"publishedDate"`
	Description         string               `json:"description"`
	ImageLinks          ImageLinks           `json:"imageLinks"`
	IndustryIdentifiers []IndustryIdentifier `json:"industryIdentifiers"`
}

type ImageLinks struct {
	Thumbnail string `json:"thumbnail"`
}

type IndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

const (
	GOOGLE_BOOKS_API = "https://www.googleapis.com/books/v1/volumes"
	API_KEY          = "AIzaSyA9LYXQu-r-FKD5WQkYUUsy2DMet6EMTPo"
)

// Track inserted ISBNs to prevent duplicates
var insertedISBNs = make(map[string]bool)

func main() {
	log.Println("Starting book seeding with quality filters...")

	dbConfig := database.Config{
		Host:     "localhost",
		Port:     5433,
		User:     "bookrate",
		Password: "localdev2178",
		DBName:   "bookrate",
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	bookRepo := database.NewBookRepository(db)

	// Load existing ISBNs to avoid duplicates
	loadExistingISBNs(db)

	totalInserted := 0
	totalSkipped := 0

	// Phase 1: Specific popular titles (get ALL editions)
	log.Println("\n=== Phase 1: Popular Titles ===")
	popularTitles := []string{
		"Harry Potter and the Philosopher's Stone",
		"Harry Potter and the Chamber of Secrets",
		"Harry Potter and the Prisoner of Azkaban",
		"Harry Potter and the Goblet of Fire",
		"Harry Potter and the Order of the Phoenix",
		"Harry Potter and the Half-Blood Prince",
		"Harry Potter and the Deathly Hallows",
		"A Game of Thrones",
		"A Clash of Kings",
		"A Storm of Swords",
		"Foundation",
		"Foundation and Empire",
		"Second Foundation",
		"Dune",
		"1984",
		"Sapiens",
		"Atomic Habits",
		"The Alchemist",
		"To Kill a Mockingbird",
		"Pride and Prejudice",
		"The Great Gatsby",
		"Thinking Fast and Slow",
		"The Lean Startup",
		"Zero to One",
	}

	for _, title := range popularTitles {
		inserted, skipped := seedByExactTitle(bookRepo, title, 10)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 2: Popular authors
	log.Println("\n=== Phase 2: Popular Authors ===")
	authors := []struct {
		name  string
		count int
	}{
		{"Stephen King", 50},
		{"J.K. Rowling", 30},
		{"George R.R. Martin", 30},
		{"Isaac Asimov", 40},
		{"Agatha Christie", 40},
		{"Dan Brown", 20},
		{"Malcolm Gladwell", 15},
		{"Yuval Noah Harari", 10},
		{"James Clear", 10},
		{"Dale Carnegie", 15},
		{"Robert Kiyosaki", 15},
		{"Ray Dalio", 10},
		{"Brandon Sanderson", 30},
		{"Neil Gaiman", 25},
		{"Terry Pratchett", 35},
		{"Margaret Atwood", 25},
	}

	for _, author := range authors {
		inserted, skipped := seedByAuthor(bookRepo, author.name, author.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 3: Bestseller categories
	log.Println("\n=== Phase 3: Bestsellers ===")
	categories := []struct {
		query string
		count int
	}{
		{"bestseller fiction 2020", 150},
		{"bestseller fiction 2021", 150},
		{"bestseller fiction 2022", 150},
		{"bestseller nonfiction 2020", 100},
		{"bestseller nonfiction 2021", 100},
		{"bestseller fantasy", 200},
		{"bestseller mystery thriller", 200},
		{"bestseller science fiction", 150},
		{"bestseller romance", 150},
		{"bestseller biography", 100},
		{"bestseller business", 150},
		{"bestseller self help", 150},
	}

	for _, cat := range categories {
		inserted, skipped := seedByQuery(bookRepo, cat.query, cat.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 4: Classic literature
	log.Println("\n=== Phase 4: Classics ===")
	classics := []struct {
		query string
		count int
	}{
		{"classic literature fiction", 300},
		{"american classics", 150},
		{"british classics", 150},
		{"russian classics", 100},
	}

	for _, classic := range classics {
		inserted, skipped := seedByQuery(bookRepo, classic.query, classic.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 5: Genre diversity
	log.Println("\n=== Phase 5: Genres ===")
	genres := []struct {
		name  string
		count int
	}{
		{"thriller", 200},
		{"horror", 150},
		{"historical fiction", 150},
		{"psychology", 150},
		{"philosophy", 100},
		{"economics", 100},
		{"technology", 100},
		{"memoir", 100},
		{"cooking", 50},
		{"health fitness", 50},
	}

	for _, genre := range genres {
		inserted, skipped := seedByQuery(bookRepo, genre.name, genre.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	log.Printf("\n=== Seeding Complete ===")
	log.Printf("Total books inserted: %d", totalInserted)
	log.Printf("Total books skipped: %d", totalSkipped)
}

func loadExistingISBNs(db *sql.DB) {
	rows, err := db.Query("SELECT isbn FROM books WHERE isbn IS NOT NULL AND isbn != ''")
	if err != nil {
		log.Printf("Warning: Could not load existing ISBNs: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var isbn string
		if err := rows.Scan(&isbn); err == nil {
			insertedISBNs[isbn] = true
			count++
		}
	}
	log.Printf("Loaded %d existing ISBNs", count)
}

func seedByExactTitle(repo *database.BookRepository, title string, maxBooks int) (int, int) {
	log.Printf("Searching for exact title: %s", title)
	return fetchAndInsert(repo, title, maxBooks, "intitle", true)
}

func seedByAuthor(repo *database.BookRepository, author string, maxBooks int) (int, int) {
	log.Printf("Searching author: %s", author)
	return fetchAndInsert(repo, author, maxBooks, "inauthor", false)
}

func seedByQuery(repo *database.BookRepository, query string, maxBooks int) (int, int) {
	log.Printf("Searching: %s", query)
	return fetchAndInsert(repo, query, maxBooks, "", false)
}

func fetchAndInsert(repo *database.BookRepository, searchTerm string, maxBooks int, searchType string, exactMatch bool) (int, int) {
	inserted := 0
	skipped := 0
	startIndex := 0
	maxResults := 40

	for inserted < maxBooks {
		var query string
		if searchType != "" {
			query = fmt.Sprintf("%s:%s", searchType, url.QueryEscape(searchTerm))
		} else {
			query = url.QueryEscape(searchTerm)
		}

		requestURL := fmt.Sprintf("%s?q=%s&startIndex=%d&maxResults=%d&key=%s",
			GOOGLE_BOOKS_API, query, startIndex, maxResults, API_KEY)

		books, err := fetchGoogleBooks(requestURL)
		if err != nil {
			log.Printf("Error: %v", err)
			break
		}

		if len(books) == 0 {
			break
		}

		for _, book := range books {
			if inserted >= maxBooks {
				break
			}

			// Quality filters
			if !isQualityBook(book.VolumeInfo, searchTerm, exactMatch) {
				skipped++
				continue
			}

			isbn := extractISBN13(book.VolumeInfo.IndustryIdentifiers)
			year := extractYear(book.VolumeInfo.PublishedDate)

			// Check for duplicate ISBN
			if isbn != "" && insertedISBNs[isbn] {
				skipped++
				continue
			}

			req := models.CreateBookRequest{
				Title:         book.VolumeInfo.Title,
				Author:        getFirstAuthor(book.VolumeInfo.Authors),
				ISBN:          isbn,
				Description:   truncateDescription(book.VolumeInfo.Description),
				PublishedYear: year,
				CoverURL:      getCoverURL(book.VolumeInfo.ImageLinks.Thumbnail),
			}

			if req.Title == "" || req.Author == "" {
				skipped++
				continue
			}

			_, err := repo.Create(req)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					if isbn != "" {
						insertedISBNs[isbn] = true
					}
					skipped++
				} else {
					skipped++
				}
				continue
			}

			// Mark ISBN as inserted
			if isbn != "" {
				insertedISBNs[isbn] = true
			}

			inserted++
			if inserted%50 == 0 {
				log.Printf("Progress: %d/%d", inserted, maxBooks)
			}

			time.Sleep(50 * time.Millisecond)
		}

		startIndex += maxResults

		// Limit pagination
		if startIndex >= 200 {
			break
		}
	}

	log.Printf("Completed '%s': %d inserted, %d skipped", searchTerm, inserted, skipped)
	return inserted, skipped
}

func isQualityBook(info VolumeInfo, searchTerm string, exactMatch bool) bool {
	title := strings.ToLower(info.Title)
	author := strings.ToLower(getFirstAuthor(info.Authors))

	// Filter out garbage
	badKeywords := []string{
		"catalogue", "catalog", "index", "proceedings", "annual report",
		"bibliography", "reference", "directory", "handbook",
		"journal of", "transactions", "bulletin", "circular",
	}

	for _, bad := range badKeywords {
		if strings.Contains(title, bad) {
			return false
		}
	}

	// Filter Unknown Author
	if author == "unknown author" || author == "" {
		return false
	}

	// Filter old books (before 1950)
	year := extractYear(info.PublishedDate)
	if year > 0 && year < 1950 {
		return false
	}

	// If exact match required, check title similarity
	if exactMatch {
		searchLower := strings.ToLower(searchTerm)
		if !strings.Contains(title, searchLower) {
			return false
		}
	}

	return true
}

func fetchGoogleBooks(requestURL string) ([]BookItem, error) {
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var booksResp GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&booksResp); err != nil {
		return nil, err
	}

	return booksResp.Items, nil
}

func extractISBN13(identifiers []IndustryIdentifier) string {
	for _, id := range identifiers {
		if id.Type == "ISBN_13" {
			return id.Identifier
		}
	}
	for _, id := range identifiers {
		if id.Type == "ISBN_10" {
			return id.Identifier
		}
	}
	return ""
}

func extractYear(publishedDate string) int {
	if len(publishedDate) >= 4 {
		var year int
		fmt.Sscanf(publishedDate[:4], "%d", &year)
		return year
	}
	return 0
}

func getFirstAuthor(authors []string) string {
	if len(authors) > 0 {
		return authors[0]
	}
	return "Unknown Author"
}

func truncateDescription(desc string) string {
	if len(desc) > 1000 {
		return desc[:997] + "..."
	}
	return desc
}

func getCoverURL(thumbnail string) string {
	if thumbnail != "" {
		return strings.Replace(thumbnail, "http://", "https://", 1)
	}
	return ""
}
