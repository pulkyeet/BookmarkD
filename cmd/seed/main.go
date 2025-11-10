package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
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
	Categories          []string             `json:"categories"`
	Language            string               `json:"language"`
	AverageRating       float64              `json:"averageRating"`
}

type ImageLinks struct {
	Thumbnail  string `json:"thumbnail"`
	SmallThumb string `json:"smallThumbnail"`
}

type IndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

const (
	GOOGLE_BOOKS_API = "https://www.googleapis.com/books/v1/volumes"
	API_KEY          = "AIzaSyA9LYXQu-r-FKD5WQkYUUsy2DMet6EMTPo"
)

// Track books to prevent duplicates
var insertedISBNs = make(map[string]bool)
var insertedTitles = make(map[string]int) // normalized_title+author -> book_id

// Genre mapping from Google Books categories to our genres
var genreMapping = map[string][]string{
	"Fiction":         {"fiction", "novel", "literature"},
	"Non-Fiction":     {"nonfiction", "non-fiction"},
	"Mystery":         {"mystery", "detective", "crime fiction"},
	"Thriller":        {"thriller", "suspense"},
	"Science Fiction": {"science fiction", "sci-fi", "scifi"},
	"Fantasy":         {"fantasy", "magic", "epic fantasy"},
	"Romance":         {"romance", "love story"},
	"Horror":          {"horror", "ghost", "supernatural"},
	"Biography":       {"biography", "memoir", "autobiography"},
	"History":         {"history", "historical"},
	"Self-Help":       {"self-help", "self help", "personal development", "motivational"},
	"Business":        {"business", "entrepreneurship", "management", "economics"},
	"Poetry":          {"poetry", "poems"},
	"Young Adult":     {"young adult", "ya fiction", "teen"},
	"Classics":        {"classics", "classic literature"},
}

func main() {
	log.Println("Starting advanced book seeding - Target: 20,000 quality books")

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
	genreRepo := database.NewGenreRepository(db)

	// Load existing books to avoid duplicates
	loadExistingBooks(db)

	totalInserted := 0
	totalSkipped := 0

	// Phase 1: Award Winners (850 total)
	log.Println("\n=== Phase 1: Award Winners ===")
	awards := []struct {
		query string
		count int
	}{
		{"Pulitzer Prize winner", 100},
		{"Booker Prize winner", 100},
		{"National Book Award winner", 100},
		{"Hugo Award winner", 100},
		{"Nebula Award winner", 75},
		{"Goodreads Choice Award winner", 200},
		{"Man Booker Prize", 75},
		{"Edgar Award winner", 75},
		{"Newbery Medal", 25},
	}

	for _, award := range awards {
		inserted, skipped := seedByQuery(bookRepo, genreRepo, award.query, award.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 2: NYT Bestsellers (3000 total)
	log.Println("\n=== Phase 2: NYT Bestsellers ===")
	for year := 2024; year >= 2015; year-- {
		queries := []string{
			fmt.Sprintf("New York Times bestseller fiction %d", year),
			fmt.Sprintf("New York Times bestseller nonfiction %d", year),
		}
		for _, q := range queries {
			inserted, skipped := seedByQuery(bookRepo, genreRepo, q, 150)
			totalInserted += inserted
			totalSkipped += skipped
		}
	}

	// Phase 3: Popular Authors (1650 total)
	log.Println("\n=== Phase 3: Popular Authors ===")
	authors := []struct {
		name  string
		count int
	}{
		{"Stephen King", 60},
		{"J.K. Rowling", 30},
		{"George R.R. Martin", 30},
		{"Brandon Sanderson", 50},
		{"Neil Gaiman", 40},
		{"Margaret Atwood", 30},
		{"Haruki Murakami", 25},
		{"Colleen Hoover", 30},
		{"James Patterson", 60},
		{"Nora Roberts", 60},
		{"Agatha Christie", 60},
		{"John Grisham", 50},
		{"Dan Brown", 20},
		{"Malcolm Gladwell", 15},
		{"Yuval Noah Harari", 10},
		{"Michelle Obama", 10},
		{"Barack Obama", 10},
		{"Taylor Jenkins Reid", 20},
		{"Delia Owens", 10},
		{"Kristin Hannah", 30},
		{"Sally Rooney", 15},
		{"Leigh Bardugo", 25},
		{"Sarah J. Maas", 40},
		{"Cassandra Clare", 40},
		{"Rick Riordan", 50},
		{"Suzanne Collins", 20},
		{"Veronica Roth", 15},
		{"Rainbow Rowell", 20},
	}

	for _, author := range authors {
		inserted, skipped := seedByAuthor(bookRepo, genreRepo, author.name, author.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 4: Subject Queries (7600 total)
	log.Println("\n=== Phase 4: Subject Queries ===")
	subjects := []struct {
		query string
		count int
	}{
		{"subject:fiction bestseller", 600},
		{"subject:mystery bestseller", 400},
		{"subject:thriller bestseller", 400},
		{"subject:fantasy bestseller", 600},
		{"subject:science fiction bestseller", 500},
		{"subject:romance bestseller", 600},
		{"subject:horror bestseller", 300},
		{"subject:young adult bestseller", 500},
		{"subject:biography bestseller", 400},
		{"subject:history bestseller", 300},
		{"subject:self-help bestseller", 400},
		{"subject:business bestseller", 400},
		{"subject:psychology bestseller", 300},
		{"subject:philosophy bestseller", 250},
		{"subject:memoir bestseller", 300},
		{"subject:true crime bestseller", 250},
		{"subject:literary fiction", 600},
		{"subject:historical fiction", 500},
		{"subject:contemporary fiction", 500},
		{"subject:dystopian fiction", 250},
		{"subject:magical realism", 150},
	}

	for _, subject := range subjects {
		inserted, skipped := seedByQuery(bookRepo, genreRepo, subject.query, subject.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 5: Popular Series (770 total)
	log.Println("\n=== Phase 5: Popular Series ===")
	series := []string{
		"Harry Potter",
		"A Song of Ice and Fire",
		"The Lord of the Rings",
		"The Hunger Games",
		"Divergent",
		"Twilight",
		"Percy Jackson",
		"The Maze Runner",
		"A Court of Thorns and Roses",
		"Throne of Glass",
		"Shadow and Bone",
		"Six of Crows",
		"The Witcher",
		"Foundation series Asimov",
		"Dune series",
		"Discworld",
		"The Expanse",
		"Outlander",
		"Jack Reacher",
		"Alex Cross",
		"Sherlock Holmes",
		"Hercule Poirot",
	}

	for _, s := range series {
		inserted, skipped := seedByQuery(bookRepo, genreRepo, s, 35)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 6: Classics (1250 total)
	log.Println("\n=== Phase 6: Classics ===")
	classics := []struct {
		query string
		count int
	}{
		{"subject:classics literature", 500},
		{"subject:american classics", 200},
		{"subject:british classics", 200},
		{"subject:russian classics", 100},
		{"subject:french classics", 100},
		{"classic novels everyone should read", 150},
	}

	for _, classic := range classics {
		inserted, skipped := seedByQuery(bookRepo, genreRepo, classic.query, classic.count)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 7: Recent Popular (2750 total)
	log.Println("\n=== Phase 7: Recent Popular Books ===")
	recentQueries := []string{
		"popular books 2024",
		"popular books 2023",
		"popular books 2022",
		"popular books 2021",
		"popular books 2020",
		"trending books 2024",
		"trending books 2023",
		"book club favorites 2024",
		"book club favorites 2023",
		"tiktok books",
		"booktok recommendations",
	}

	for _, q := range recentQueries {
		inserted, skipped := seedByQuery(bookRepo, genreRepo, q, 250)
		totalInserted += inserted
		totalSkipped += skipped
	}

	// Phase 8: Quality Fill (to reach 20K)
	log.Println("\n=== Phase 8: Quality Fill ===")
	remaining := 20000 - totalInserted
	if remaining > 0 {
		log.Printf("Filling remaining %d books with quality searches...", remaining)
		fillQueries := []string{
			"highly rated fiction",
			"highly rated nonfiction",
			"award winning books",
			"critically acclaimed books",
			"must read books",
		}
		perQuery := remaining / len(fillQueries)
		for _, q := range fillQueries {
			inserted, skipped := seedByQuery(bookRepo, genreRepo, q, perQuery)
			totalInserted += inserted
			totalSkipped += skipped
			if totalInserted >= 20000 {
				break
			}
		}
	}

	log.Printf("\n=== Seeding Complete ===")
	log.Printf("Total books inserted: %d", totalInserted)
	log.Printf("Total books skipped: %d", totalSkipped)
	log.Printf("Final book count target: 20,000")
}

func loadExistingBooks(db *sql.DB) {
	// Load ISBNs
	rows, err := db.Query("SELECT isbn FROM books WHERE isbn IS NOT NULL AND isbn != ''")
	if err != nil {
		log.Printf("Warning: Could not load existing ISBNs: %v", err)
	} else {
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

	// Load title+author combinations
	rows2, err := db.Query("SELECT id, title, author FROM books")
	if err != nil {
		log.Printf("Warning: Could not load existing books: %v", err)
		return
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		var id int
		var title, author string
		if err := rows2.Scan(&id, &title, &author); err == nil {
			key := normalizeBookKey(title, author)
			insertedTitles[key] = id
			count++
		}
	}
	log.Printf("Loaded %d existing title+author combinations", count)
}

func normalizeBookKey(title, author string) string {
	// Normalize title: lowercase, remove articles, punctuation, extra spaces
	title = strings.ToLower(title)
	title = strings.TrimSpace(title)

	// Remove leading articles
	title = regexp.MustCompile(`^(the|a|an)\s+`).ReplaceAllString(title, "")

	// Remove subtitles (everything after : or -)
	if idx := strings.Index(title, ":"); idx > 0 {
		title = title[:idx]
	}
	if idx := strings.Index(title, " - "); idx > 0 {
		title = title[:idx]
	}

	// Remove all punctuation and extra spaces
	title = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(title, "")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Normalize author
	author = strings.ToLower(strings.TrimSpace(author))
	author = regexp.MustCompile(`\s+`).ReplaceAllString(author, " ")

	return title + "|" + author
}

func seedByAuthor(repo *database.BookRepository, genreRepo *database.GenreRepository, author string, maxBooks int) (int, int) {
	log.Printf("Searching author: %s", author)
	return fetchAndInsert(repo, genreRepo, author, maxBooks, "inauthor")
}

func seedByQuery(repo *database.BookRepository, genreRepo *database.GenreRepository, query string, maxBooks int) (int, int) {
	log.Printf("Searching: %s", query)
	return fetchAndInsert(repo, genreRepo, query, maxBooks, "")
}

func fetchAndInsert(repo *database.BookRepository, genreRepo *database.GenreRepository, searchTerm string, maxBooks int, searchType string) (int, int) {
	inserted := 0
	skipped := 0
	startIndex := 0
	maxResults := 40
	maxAttempts := maxBooks * 5 // Process up to 5x target to account for skips

	processedCount := 0

	for inserted < maxBooks && processedCount < maxAttempts {
		var query string
		if searchType != "" {
			query = fmt.Sprintf("%s:%s", searchType, url.QueryEscape(searchTerm))
		} else {
			query = url.QueryEscape(searchTerm)
		}

		requestURL := fmt.Sprintf("%s?q=%s&startIndex=%d&maxResults=%d&key=%s&orderBy=relevance",
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
			processedCount++

			if inserted >= maxBooks {
				break
			}

			// Strict quality filters
			if !isQualityBook(book.VolumeInfo) {
				skipped++
				continue
			}

			isbn := extractISBN13(book.VolumeInfo.IndustryIdentifiers)
			year := extractYear(book.VolumeInfo.PublishedDate)
			author := getFirstAuthor(book.VolumeInfo.Authors)

			// Check for duplicate ISBN
			if isbn != "" && insertedISBNs[isbn] {
				skipped++
				continue
			}

			// Check for duplicate title+author
			bookKey := normalizeBookKey(book.VolumeInfo.Title, author)
			if _, exists := insertedTitles[bookKey]; exists {
				skipped++
				continue
			}

			req := models.CreateBookRequest{
				Title:         book.VolumeInfo.Title,
				Author:        author,
				ISBN:          isbn,
				Description:   truncateDescription(book.VolumeInfo.Description),
				PublishedYear: year,
				CoverURL:      getCoverURL(book.VolumeInfo.ImageLinks),
			}

			createdBook, err := repo.Create(req)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					if isbn != "" {
						insertedISBNs[isbn] = true
					}
					insertedTitles[bookKey] = 0
				}
				skipped++
				continue
			}

			// Mark as inserted
			if isbn != "" {
				insertedISBNs[isbn] = true
			}
			insertedTitles[bookKey] = createdBook.ID

			// Assign genres
			assignGenres(genreRepo, createdBook.ID, book.VolumeInfo.Categories)

			inserted++
			if inserted%100 == 0 {
				log.Printf("Progress: %d/%d (skipped: %d)", inserted, maxBooks, skipped)
			}

			time.Sleep(100 * time.Millisecond)
		}

		startIndex += maxResults

		// Safety valve - stop if we've processed 5x the target
		if processedCount >= maxAttempts {
			log.Printf("Reached max attempts (%d processed) for '%s'", processedCount, searchTerm)
			break
		}
	}

	log.Printf("Completed '%s': %d inserted, %d skipped (processed %d total)", searchTerm, inserted, skipped, processedCount)
	return inserted, skipped
}

func isQualityBook(info VolumeInfo) bool {
	title := strings.ToLower(info.Title)
	author := strings.ToLower(getFirstAuthor(info.Authors))

	// MUST have description (50+ chars, not 100)
	if len(strings.TrimSpace(info.Description)) < 50 {
		return false
	}

	// MUST have cover image
	if info.ImageLinks.Thumbnail == "" && info.ImageLinks.SmallThumb == "" {
		return false
	}

	// MUST have ISBN
	isbn := extractISBN13(info.IndustryIdentifiers)
	if isbn == "" {
		return false
	}

	// MUST be English
	if info.Language != "" && info.Language != "en" {
		return false
	}

	// Filter out garbage keywords
	badKeywords := []string{
		"catalogue", "catalog", "index", "proceedings", "annual report",
		"bibliography", "reference", "directory", "handbook",
		"journal of", "transactions", "bulletin", "circular",
		"workbook", "study guide", "teacher edition", "student edition",
		"test prep", "exam", "textbook", "course",
	}

	for _, bad := range badKeywords {
		if strings.Contains(title, bad) {
			return false
		}
	}

	// Filter Unknown Author
	if author == "unknown author" || author == "" || author == "various" || author == "anonymous" {
		return false
	}

	// Year range: 1950-2025
	year := extractYear(info.PublishedDate)
	if year > 0 && (year < 1950 || year > 2025) {
		return false
	}

	return true
}

func assignGenres(genreRepo *database.GenreRepository, bookID int, categories []string) {
	if len(categories) == 0 {
		return
	}

	assignedGenres := make(map[string]bool)

	for _, category := range categories {
		catLower := strings.ToLower(category)

		// Match against our genre mapping
		for genreName, keywords := range genreMapping {
			if assignedGenres[genreName] {
				continue
			}

			for _, keyword := range keywords {
				if strings.Contains(catLower, keyword) {
					// Get genre ID
					genre, err := genreRepo.GetByName(genreName)
					if err != nil || genre == nil {
						continue
					}

					// Add to book_genres
					genreRepo.AddGenreToBook(bookID, genre.ID)
					assignedGenres[genreName] = true
					break
				}
			}
		}
	}
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

func getCoverURL(imageLinks ImageLinks) string {
	thumbnail := imageLinks.Thumbnail
	if thumbnail == "" {
		thumbnail = imageLinks.SmallThumb
	}
	if thumbnail != "" {
		return strings.Replace(thumbnail, "http://", "https://", 1)
	}
	return ""
}
