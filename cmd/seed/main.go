package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/pulkyeet/BookmarkD/internal/database"
	"github.com/pulkyeet/BookmarkD/internal/models"
)

// ============================================================
// CONFIGURATION - Tweak these to control seeding behavior
// ============================================================

const (
	TARGET_BOOKS = 25000 // Total books to aim for. 25K fits comfortably in 1GB Fly.io disk (~50MB).
	// RUNTIME NOTE: Single run yields ~15-18K books (6-10 hours depending on rate limits).
	// Script is RE-RUNNABLE: dedup prevents duplicates, so run again next day to reach 25K.
	// Google Books caps at ~1000 requests/day free. With 250ms delay, runs for several hours.

	// Rate limiting: Google Books free tier = ~1000 requests/day, 40 results each.
	// Previous 429 errors were from speed, not daily cap. 250ms is safe.
	BASE_DELAY_MS     = 250  // Milliseconds between API requests
	BACKOFF_INITIAL_S = 30   // Seconds to wait on first 429
	BACKOFF_MAX_S     = 300  // Max backoff seconds
	MAX_RETRIES       = 5    // Retries per request before giving up
	MAX_RESULTS       = 40   // Google Books max per page

	// Quality filters
	MIN_DESCRIPTION_LEN = 50   // Chars minimum for description
	MIN_YEAR            = 1800 // Catches classics (Pride & Prejudice 1813, etc.)
	MAX_YEAR            = 2026 // Current year

	// Google Books API
	GOOGLE_BOOKS_API = "https://www.googleapis.com/books/v1/volumes"
)

// API key: checks env var first, falls back to hardcoded.
// Usage: GOOGLE_BOOKS_API_KEY="xxx" go run cmd/seed/main.go
func getAPIKey() string {
	if key := os.Getenv("GOOGLE_BOOKS_API_KEY"); key != "" {
		return key
	}
	return "AIzaSyA9LYXQu-r-FKD5WQkYUUsy2DMet6EMTPo"
}

// ============================================================
// DATA TYPES
// ============================================================

type GoogleBooksResponse struct {
	TotalItems int        `json:"totalItems"`
	Items      []BookItem `json:"items"`
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
	PageCount           int                  `json:"pageCount"`
}

type ImageLinks struct {
	Thumbnail  string `json:"thumbnail"`
	SmallThumb string `json:"smallThumbnail"`
}

type IndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

// ============================================================
// DEDUPLICATION STATE
// ============================================================

var insertedISBNs = make(map[string]bool)
var insertedTitles = make(map[string]int) // normalizedKey -> bookID

// ============================================================
// GENRE MAPPING (Google Books categories -> our genres)
// ============================================================

var genreMapping = map[string][]string{
	"Fiction":         {"fiction", "novel", "literature", "literary fiction"},
	"Non-Fiction":     {"nonfiction", "non-fiction"},
	"Mystery":         {"mystery", "detective", "crime fiction", "whodunit"},
	"Thriller":        {"thriller", "suspense", "espionage"},
	"Science Fiction": {"science fiction", "sci-fi", "scifi", "space opera"},
	"Fantasy":         {"fantasy", "magic", "epic fantasy", "urban fantasy", "dark fantasy"},
	"Romance":         {"romance", "love story", "romantic", "contemporary romance"},
	"Horror":          {"horror", "ghost", "supernatural horror", "gothic"},
	"Biography":       {"biography", "memoir", "autobiography", "life story"},
	"History":         {"history", "historical", "military history", "world history"},
	"Self-Help":       {"self-help", "self help", "personal development", "motivational", "self-improvement"},
	"Business":        {"business", "entrepreneurship", "management", "economics", "finance", "investing"},
	"Poetry":          {"poetry", "poems", "verse"},
	"Young Adult":     {"young adult", "ya fiction", "teen", "coming of age"},
	"Classics":        {"classics", "classic literature", "classic fiction"},
}

// ============================================================
// SEARCH CONFIGURATION
// Each section is a phase. Counts are targets (not guaranteed).
// Adjust counts freely; script handles deduplication.
// ============================================================

type queryConfig struct {
	Query string
	Count int
}

type authorConfig struct {
	Name  string
	Count int
}

// --- PHASE 1: AWARD WINNERS (~900 books) ---
// Most reliable quality signal. Award-winning = people read them.
var awardQueries = []queryConfig{
	{"Pulitzer Prize fiction", 100},
	{"Pulitzer Prize nonfiction", 75},
	{"Booker Prize winner", 100},
	{"National Book Award winner", 100},
	{"Hugo Award winner", 100},
	{"Nebula Award winner", 75},
	{"Goodreads Choice Award winner 2024", 100},
	{"Goodreads Choice Award winner 2023", 100},
	{"Man Booker International Prize", 50},
	{"Edgar Award mystery", 50},
	{"Costa Book Award", 50},
}

// --- PHASE 2: NYT BESTSELLERS (~2500 books) ---
// Year-by-year ensures we get the actual bestsellers, not random results.
// Generates queries dynamically in main().

// --- PHASE 3: POPULAR AUTHORS (~8000 books) ---
// THE MOST RELIABLE PHASE. Author queries return their actual books.
// This is the backbone of the catalog.
var popularAuthors = []authorConfig{
	// === LITERARY FICTION ===
	{"Margaret Atwood", 35},
	{"Haruki Murakami", 25},
	{"Kazuo Ishiguro", 15},
	{"Toni Morrison", 20},
	{"Chimamanda Ngozi Adichie", 15},
	{"Zadie Smith", 15},
	{"Donna Tartt", 10},
	{"Khaled Hosseini", 10},
	{"Celeste Ng", 10},
	{"Hanya Yanagihara", 5},
	{"Jhumpa Lahiri", 10},
	{"Salman Rushdie", 20},
	{"Ian McEwan", 20},
	{"Hilary Mantel", 15},
	{"Cormac McCarthy", 15},
	{"Jonathan Franzen", 10},
	{"Don DeLillo", 15},
	{"Arundhati Roy", 5},
	{"Elena Ferrante", 10},
	{"Amor Towles", 5},
	{"Anthony Doerr", 5},
	{"Madeline Miller", 5},

	// === THRILLER / MYSTERY / CRIME ===
	{"James Patterson", 60},
	{"John Grisham", 50},
	{"Lee Child", 40},
	{"Michael Connelly", 40},
	{"Karin Slaughter", 30},
	{"Tana French", 15},
	{"Gillian Flynn", 5},
	{"Paula Hawkins", 5},
	{"Ruth Ware", 15},
	{"Lisa Jewell", 15},
	{"Freida McFadden", 15},
	{"Riley Sager", 10},
	{"Harlan Coben", 40},
	{"Daniel Silva", 30},
	{"David Baldacci", 40},
	{"Dan Brown", 10},
	{"Stieg Larsson", 5},
	{"Jo Nesbo", 15},
	{"Tess Gerritsen", 25},
	{"Linwood Barclay", 15},
	{"Alex Michaelides", 5},
	{"Delia Owens", 5},

	// === ROMANCE / CONTEMPORARY ===
	{"Colleen Hoover", 20},
	{"Emily Henry", 10},
	{"Ali Hazelwood", 10},
	{"Christina Lauren", 15},
	{"Abby Jimenez", 10},
	{"Lynn Painter", 5},
	{"Ana Huang", 10},
	{"Penelope Douglas", 10},
	{"Elle Kennedy", 10},
	{"Nora Roberts", 60},
	{"Nicholas Sparks", 25},
	{"Jojo Moyes", 15},
	{"Sally Rooney", 5},
	{"Taylor Jenkins Reid", 10},
	{"Kristin Hannah", 20},
	{"Lisa Kleypas", 20},
	{"Julia Quinn", 15},
	{"Jasmine Guillory", 10},
	{"Talia Hibbert", 10},
	{"Helen Hoang", 5},
	{"Beth O'Leary", 10},
	{"Marian Keyes", 15},
	{"Liane Moriarty", 15},
	{"Jodi Picoult", 30},

	// === FANTASY / SCI-FI ===
	{"Brandon Sanderson", 50},
	{"George R.R. Martin", 25},
	{"Neil Gaiman", 30},
	{"Sarah J. Maas", 25},
	{"Leigh Bardugo", 15},
	{"V.E. Schwab", 15},
	{"Rebecca Yarros", 5},
	{"Holly Black", 15},
	{"Patrick Rothfuss", 5},
	{"Terry Pratchett", 50},
	{"Ursula K. Le Guin", 30},
	{"Andy Weir", 5},
	{"Blake Crouch", 10},
	{"R.F. Kuang", 5},
	{"Joe Abercrombie", 15},
	{"Robin Hobb", 25},
	{"Scott Lynch", 5},
	{"N.K. Jemisin", 10},
	{"Naomi Novik", 10},
	{"Samantha Shannon", 5},
	{"Pierce Brown", 5},
	{"Adrian Tchaikovsky", 15},
	{"Martha Wells", 10},
	{"Travis Baldree", 5},
	{"T.J. Klune", 5},
	{"Becky Chambers", 5},
	{"Philip Pullman", 15},

	// === HORROR ===
	{"Stephen King", 70},
	{"Paul Tremblay", 10},
	{"Grady Hendrix", 10},
	{"Silvia Moreno-Garcia", 10},
	{"Shirley Jackson", 10},
	{"Peter Straub", 10},
	{"Joe Hill", 15},
	{"Dean Koontz", 40},
	{"Clive Barker", 15},

	// === HISTORICAL FICTION ===
	{"Ken Follett", 30},
	{"Lisa See", 15},
	{"Kate Quinn", 10},
	{"Philippa Gregory", 25},
	{"Tracy Chevalier", 10},
	{"Chris Bohjalian", 15},
	{"Geraldine Brooks", 10},
	{"Tatiana de Rosnay", 5},

	// === YA / CHILDREN'S ===
	{"J.K. Rowling", 20},
	{"Rick Riordan", 40},
	{"Suzanne Collins", 10},
	{"Veronica Roth", 10},
	{"Cassandra Clare", 30},
	{"Rainbow Rowell", 10},
	{"John Green", 10},
	{"Angie Thomas", 5},
	{"Becky Albertalli", 5},
	{"Adam Silvera", 5},
	{"Jason Reynolds", 10},
	{"Jeff Kinney", 15},
	{"Dav Pilkey", 10},
	{"Roald Dahl", 20},
	{"R.L. Stine", 30},

	// === NON-FICTION: BUSINESS / SELF-HELP ===
	{"James Clear", 5},
	{"Mark Manson", 5},
	{"Brene Brown", 15},
	{"Adam Grant", 10},
	{"Cal Newport", 10},
	{"Morgan Housel", 5},
	{"Robert Kiyosaki", 10},
	{"Tim Ferriss", 5},
	{"Ryan Holiday", 15},
	{"Simon Sinek", 5},
	{"Daniel Kahneman", 5},
	{"Nassim Nicholas Taleb", 5},
	{"Ray Dalio", 5},
	{"Angela Duckworth", 5},
	{"Carol Dweck", 5},
	{"Stephen Covey", 5},
	{"Dale Carnegie", 5},
	{"Robert Greene", 5},
	{"Seth Godin", 10},
	{"Peter Thiel", 5},
	{"Eric Ries", 5},
	{"Jim Collins", 5},
	{"Clayton Christensen", 5},

	// === NON-FICTION: SCIENCE / POPULAR SCIENCE ===
	{"Yuval Noah Harari", 5},
	{"Bill Bryson", 20},
	{"Malcolm Gladwell", 10},
	{"Mary Roach", 10},
	{"Neil deGrasse Tyson", 10},
	{"Oliver Sacks", 15},
	{"Steven Pinker", 10},
	{"Jared Diamond", 5},
	{"Richard Dawkins", 10},
	{"Stephen Hawking", 10},
	{"Michio Kaku", 10},
	{"Robert Sapolsky", 5},
	{"Siddhartha Mukherjee", 5},
	{"Atul Gawande", 5},
	{"Carl Sagan", 10},

	// === NON-FICTION: MEMOIR / BIOGRAPHY ===
	{"Michelle Obama", 5},
	{"Barack Obama", 5},
	{"Tara Westover", 5},
	{"Trevor Noah", 5},
	{"Walter Isaacson", 10},
	{"Ashlee Vance", 5},
	{"Michael Lewis", 15},
	{"Jon Krakauer", 10},
	{"David Sedaris", 15},
	{"Cheryl Strayed", 5},
	{"Glennon Doyle", 5},
	{"Matthew McConaughey", 5},
	{"Andre Agassi", 5},
	{"Jennette McCurdy", 5},
	{"Viola Davis", 5},

	// === NON-FICTION: HISTORY / POLITICS ===
	{"Doris Kearns Goodwin", 10},
	{"Erik Larson", 10},
	{"David McCullough", 15},
	{"Ron Chernow", 10},
	{"Timothy Snyder", 5},
	{"Howard Zinn", 5},
	{"Ibram X. Kendi", 5},
	{"Ta-Nehisi Coates", 5},
	{"Isabel Wilkerson", 5},

	// === NON-FICTION: PSYCHOLOGY / PHILOSOPHY ===
	{"Gabor Mate", 5},
	{"Bessel van der Kolk", 5},
	{"Jordan Peterson", 5},
	{"Viktor Frankl", 5},
	{"Irvin Yalom", 10},
	{"Alain de Botton", 10},

	// === CLASSICS AUTHORS ===
	{"Jane Austen", 10},
	{"Charles Dickens", 20},
	{"F. Scott Fitzgerald", 5},
	{"George Orwell", 10},
	{"Ernest Hemingway", 15},
	{"Mark Twain", 15},
	{"Leo Tolstoy", 10},
	{"Fyodor Dostoevsky", 10},
	{"Virginia Woolf", 10},
	{"Gabriel Garcia Marquez", 10},
	{"Franz Kafka", 10},
	{"Herman Melville", 5},
	{"Oscar Wilde", 10},
	{"Emily Bronte", 5},
	{"Charlotte Bronte", 5},
	{"Harper Lee", 5},
	{"J.R.R. Tolkien", 15},
	{"C.S. Lewis", 15},
	{"Kurt Vonnegut", 15},
	{"Ray Bradbury", 15},
	{"Isaac Asimov", 30},
	{"Arthur C. Clarke", 15},
	{"Philip K. Dick", 20},
	{"Aldous Huxley", 5},
	{"Albert Camus", 5},
	{"Hermann Hesse", 5},
	{"Paulo Coelho", 10},
	{"Agatha Christie", 60},

	// === ADDITIONAL POPULAR (millions sold, widely read) ===
	{"Danielle Steel", 50},
	{"Janet Evanovich", 30},
	{"Diana Gabaldon", 15},
	{"Clive Cussler", 30},
	{"Tom Clancy", 30},
	{"Robin Cook", 20},
	{"Mary Higgins Clark", 30},
	{"Sandra Brown", 30},
	{"Stuart Woods", 30},
	{"Vince Flynn", 20},
	{"Brad Thor", 15},
	{"Catherine Coulter", 20},
	{"Debbie Macomber", 25},
	{"Elin Hilderbrand", 20},
	{"Dorothea Benton Frank", 10},
	{"Kristin Hannah", 5}, // might dup but dedup handles it
	{"Kate Morton", 10},
	{"Mitch Albom", 10},
	{"Paulo Coelho", 5}, // might dup
	{"Khaled Hosseini", 5}, // might dup
	{"Fredrik Backman", 10},
	{"Matt Haig", 10},
	{"Bonnie Garmus", 5},
	{"Brit Bennett", 5},
	{"Delia Owens", 5},
	{"Ottessa Moshfegh", 5},
	{"Carmen Maria Machado", 5},
	{"Ocean Vuong", 5},
	{"Gabrielle Zevin", 5},
	{"Delia Ephron", 5},
	{"Ann Patchett", 15},
	{"Elizabeth Strout", 10},
	{"Richard Powers", 5},
	{"Jennifer Egan", 5},
	{"Rachel Kushner", 5},
	{"George Saunders", 10},
	{"Viet Thanh Nguyen", 5},
	{"Min Jin Lee", 5},
	{"Abraham Verghese", 5},
	{"Kevin Kwan", 5},
	{"Celeste Ng", 5},

	// === ADDITIONAL NON-FICTION ===
	{"Malcolm X", 5},
	{"Angela Davis", 5},
	{"bell hooks", 10},
	{"Rebecca Solnit", 10},
	{"Roxane Gay", 10},
	{"Maggie Nelson", 5},
	{"Jenny Odell", 5},
	{"Annie Dillard", 5},
	{"Joan Didion", 10},
	{"Susan Sontag", 5},
	{"Hannah Arendt", 5},
	{"Noam Chomsky", 15},
	{"Howard Zinn", 5},
	{"Naomi Klein", 10},
	{"Matthew Walker", 5},
	{"Andrew Huberman", 5},
	{"Peter Attia", 5},
	{"James Nestor", 5},
	{"Johann Hari", 5},
	{"Dan Ariely", 5},
	{"Chip Heath", 5},
	{"Charles Duhigg", 5},
	{"Gretchen Rubin", 5},
	{"BJ Fogg", 5},
	{"Nir Eyal", 5},
	{"Ben Horowitz", 5},
	{"Reid Hoffman", 5},
	{"Satya Nadella", 5},
	{"Phil Knight", 5},
	{"Howard Schultz", 5},
	{"Ed Catmull", 5},
	{"Daniel Pink", 10},
	{"Mihaly Csikszentmihalyi", 5},
	{"Martin Seligman", 5},
	{"David Goggins", 5},
	{"Jocko Willink", 5},
}

// --- PHASE 4: SUBJECT QUERIES (~5000 books) ---
// Google Books subject: filter is more precise than free-text.
// These catch books not covered by specific author searches.
var subjectQueries = []queryConfig{
	// Fiction subgenres (widest net for popular books)
	{"subject:fiction bestseller", 500},
	{"subject:literary fiction", 500},
	{"subject:historical fiction", 500},
	{"subject:contemporary fiction", 400},
	{"subject:dystopian fiction", 200},
	{"subject:magical realism", 150},
	{"subject:women's fiction", 400},
	{"subject:domestic fiction", 200},
	{"subject:war fiction", 150},
	{"subject:satirical fiction", 100},
	{"subject:short stories collection", 150},
	{"subject:family saga", 150},
	{"subject:coming of age fiction", 200},
	{"subject:southern fiction", 100},
	{"subject:gothic fiction", 100},

	// Genre fiction
	{"subject:mystery fiction", 500},
	{"subject:thriller fiction", 500},
	{"subject:science fiction", 500},
	{"subject:fantasy fiction", 500},
	{"subject:romance fiction", 500},
	{"subject:horror fiction", 300},
	{"subject:adventure fiction", 300},
	{"subject:crime fiction", 400},
	{"subject:spy fiction", 150},
	{"subject:psychological thriller", 300},
	{"subject:cozy mystery", 200},
	{"subject:urban fantasy", 150},
	{"subject:space opera", 150},
	{"subject:paranormal romance", 150},
	{"subject:romantic suspense", 200},
	{"subject:police procedural", 150},
	{"subject:legal thriller", 100},
	{"subject:medical thriller", 100},
	{"subject:military science fiction", 100},
	{"subject:dark romance", 100},
	{"subject:fantasy romance", 150},
	{"subject:epic fantasy", 200},

	// Non-fiction
	{"subject:biography", 400},
	{"subject:memoir", 400},
	{"subject:true crime", 300},
	{"subject:popular science", 300},
	{"subject:psychology", 300},
	{"subject:philosophy", 200},
	{"subject:business economics", 300},
	{"subject:self-help", 400},
	{"subject:health fitness", 200},
	{"subject:cooking", 200},
	{"subject:travel writing", 100},
	{"subject:political science", 200},
	{"subject:sociology", 100},
	{"subject:technology", 100},
	{"subject:environmental", 100},
	{"subject:parenting", 100},
	{"subject:education", 100},
	{"subject:art", 100},
	{"subject:music", 100},
	{"subject:sports", 150},
	{"subject:nature", 100},
	{"subject:religion spirituality", 200},

	// YA subgenres
	{"subject:young adult fiction", 500},
	{"subject:young adult fantasy", 300},
	{"subject:young adult romance", 200},
	{"subject:children's fiction", 300},
	{"subject:middle grade fiction", 200},
}

// --- PHASE 5: POPULAR SERIES (~1000 books) ---
var seriesQueries = []queryConfig{
	// Fantasy/Sci-Fi
	{"Harry Potter series", 10},
	{"A Song of Ice and Fire Martin", 10},
	{"Lord of the Rings Tolkien", 10},
	{"Dune Frank Herbert", 10},
	{"Foundation Asimov", 10},
	{"Discworld Pratchett", 50},
	{"The Expanse James Corey", 10},
	{"Stormlight Archive Sanderson", 10},
	{"Mistborn Sanderson", 10},
	{"Wheel of Time Robert Jordan", 15},
	{"Kingkiller Chronicle Rothfuss", 5},
	{"First Law Abercrombie", 10},
	{"Broken Earth Jemisin", 5},
	{"Realm of the Elderlings Hobb", 15},
	{"Gentleman Bastard Lynch", 5},
	{"Red Rising Pierce Brown", 5},
	{"Hitchhiker's Guide Galaxy Adams", 5},
	{"Ender's Game series Card", 10},

	// Romance/Contemporary Fantasy
	{"A Court of Thorns and Roses Maas", 10},
	{"Throne of Glass Maas", 10},
	{"Shadow and Bone Bardugo", 10},
	{"Six of Crows Bardugo", 5},
	{"Fourth Wing Yarros", 5},
	{"The Folk of the Air Black", 5},
	{"Shatter Me Mafi", 10},
	{"Bridgerton Quinn", 10},

	// Mystery/Thriller
	{"Jack Reacher Lee Child", 30},
	{"Alex Cross Patterson", 30},
	{"Sherlock Holmes Doyle", 15},
	{"Hercule Poirot Christie", 30},
	{"Miss Marple Christie", 15},
	{"Harry Bosch Connelly", 25},
	{"Millennium Larsson Lagercrantz", 5},
	{"The Witcher Sapkowski", 10},
	{"Outlander Gabaldon", 10},

	// YA
	{"Hunger Games Collins", 5},
	{"Divergent Roth", 5},
	{"Percy Jackson Riordan", 15},
	{"Maze Runner Dashner", 5},
	{"Mortal Instruments Clare", 10},
	{"Twilight Meyer", 5},
	{"Diary of a Wimpy Kid Kinney", 15},
	{"Captain Underpants Pilkey", 10},

	// Non-fiction series
	{"Sapiens Harari", 5},
	{"Freakonomics Levitt", 5},
}

// --- PHASE 6: TRENDING / BOOKTOK / RECENT / DISCOVERY (~5000 books) ---
var trendingQueries = []queryConfig{
	// BookTok / social media hits
	{"booktok popular 2024", 200},
	{"booktok popular 2025", 200},
	{"booktok recommendations", 200},
	{"tiktok books trending", 150},
	{"bookstagram popular", 100},
	{"goodreads popular 2024", 200},
	{"goodreads popular 2023", 200},
	{"goodreads popular 2022", 150},

	// Recent bestsellers by year
	{"bestseller fiction 2025", 200},
	{"bestseller nonfiction 2025", 150},
	{"bestseller fiction 2024", 200},
	{"bestseller nonfiction 2024", 150},
	{"bestseller fiction 2023", 200},
	{"bestseller nonfiction 2023", 150},
	{"bestseller fiction 2022", 200},
	{"bestseller nonfiction 2022", 150},
	{"bestseller fiction 2021", 150},
	{"bestseller fiction 2020", 150},
	{"bestseller fiction 2019", 150},
	{"bestseller fiction 2018", 150},

	// Book club favorites
	{"book club favorites 2024", 200},
	{"book club picks 2023", 150},
	{"book club picks 2024", 150},
	{"Reese book club picks", 150},
	{"Oprah book club picks", 150},
	{"celebrity book club", 100},
	{"Barnes Noble book club", 100},

	// Award nominees (not just winners)
	{"National Book Award finalist", 150},
	{"Booker Prize longlist", 150},
	{"Pulitzer finalist", 100},
	{"Hugo Award finalist", 100},
	{"Women's Prize fiction", 100},
	{"Andrew Carnegie Medal", 50},

	// Discovery queries
	{"most anticipated books 2025", 150},
	{"most anticipated books 2024", 150},
	{"debut novels 2024", 150},
	{"debut novels 2023", 100},
	{"indie bookstore favorites", 100},
	{"librarian recommended books", 100},
	{"staff picks bookstore", 100},

	// Specific viral/popular titles (title searches for must-haves)
	{"Atomic Habits James Clear", 5},
	{"The Body Keeps the Score", 5},
	{"Educated Tara Westover", 5},
	{"Where the Crawdads Sing", 5},
	{"It Ends with Us Hoover", 5},
	{"The Seven Husbands of Evelyn Hugo", 5},
	{"Tomorrow and Tomorrow and Tomorrow", 5},
	{"Lessons in Chemistry Garmus", 5},
	{"Project Hail Mary Weir", 5},
	{"The Midnight Library Haig", 5},
	{"A Little Life Yanagihara", 5},
	{"Normal People Rooney", 5},
	{"The Song of Achilles Miller", 5},
	{"Circe Madeline Miller", 5},
	{"Daisy Jones and the Six", 5},
	{"Verity Colleen Hoover", 5},
	{"The Invisible Life of Addie LaRue", 5},
	{"Beach Read Emily Henry", 5},
	{"People We Meet on Vacation", 5},
	{"Happy Place Emily Henry", 5},
	{"Book Lovers Emily Henry", 5},
	{"The Love Hypothesis Hazelwood", 5},
	{"Fourth Wing Rebecca Yarros", 5},
	{"Iron Flame Rebecca Yarros", 5},
	{"Powerless Lauren Roberts", 5},
	{"A Good Girl's Guide to Murder", 5},
	{"House of Salt and Sorrows", 5},
	{"The Cruel Prince Black", 5},
	{"Babel R.F. Kuang", 5},
	{"The Poppy War Kuang", 5},
	{"Piranesi Susanna Clarke", 5},
	{"Klara and the Sun Ishiguro", 5},
	{"The Lincoln Highway Towles", 5},
	{"A Gentleman in Moscow Towles", 5},
	{"The Goldfinch Donna Tartt", 5},
	{"Demon Copperhead Kingsolver", 5},
	{"Trust Hernan Diaz", 5},
	{"All the Light We Cannot See", 5},
	{"The House in the Cerulean Sea", 5},
	{"Anxious People Fredrik Backman", 5},
	{"Mexican Gothic Moreno-Garcia", 5},
	{"The Vanishing Half Bennett", 5},
	{"Hamnet Maggie O'Farrell", 5},
	{"Shuggie Bain Douglas Stuart", 5},
	{"The Personal Librarian", 5},
	{"Malibu Rising Jenkins Reid", 5},
	{"Cloud Cuckoo Land Doerr", 5},
	{"Sea of Tranquility Mandel", 5},
	{"Yellowface Kuang", 5},
	{"Holly Stephen King", 5},
	{"The Covenant of Water Verghese", 5},
	{"Tom Lake Ann Patchett", 5},
	{"North Woods Daniel Mason", 5},
	{"Intermezzo Sally Rooney", 5},
	{"James Percival Everett", 5},
	{"Same As It Ever Was Claire Lombardo", 5},
}

// --- PHASE 7: CLASSICS (~1500 books) ---
var classicQueries = []queryConfig{
	{"subject:classics literature", 400},
	{"subject:american classics", 200},
	{"subject:british classics", 200},
	{"subject:russian literature classics", 150},
	{"subject:french literature classics", 100},
	{"subject:japanese literature", 75},
	{"subject:latin american literature", 75},
	{"subject:african literature", 50},
	{"subject:indian literature english", 50},
	{"classic novels must read", 300},
	{"greatest novels all time", 300},
	{"100 best books ever written", 200},
	{"modern classics fiction", 200},
	{"20th century classics literature", 200},
	{"19th century classics literature", 150},
	{"postmodern literature", 100},
	{"beat generation literature", 50},
	{"existentialist literature", 50},
}

// ============================================================
// MAIN
// ============================================================

func main() {
	log.Println("=== BookmarkD Seeder ===")
	log.Printf("Target: %d quality books", TARGET_BOOKS)
	log.Printf("Year range: %d-%d", MIN_YEAR, MAX_YEAR)
	log.Printf("Rate limit: %dms between requests", BASE_DELAY_MS)

	db := connectDB()
	defer db.Close()

	bookRepo := database.NewBookRepository(db)
	genreRepo := database.NewGenreRepository(db)

	// Load existing books for dedup
	loadExistingBooks(db)
	log.Printf("Existing catalog: %d ISBNs, %d title+author combos", len(insertedISBNs), len(insertedTitles))

	totalInserted := 0
	totalSkipped := 0

	// --- PHASE 1: Award Winners ---
	log.Println("\n========== PHASE 1: Award Winners ==========")
	for _, q := range awardQueries {
		ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
		totalInserted += ins
		totalSkipped += skip
		logProgress(totalInserted, TARGET_BOOKS)
	}

	// --- PHASE 2: NYT Bestsellers (generated dynamically) ---
	log.Println("\n========== PHASE 2: NYT Bestsellers ==========")
	for year := 2025; year >= 2005; year-- {
		for _, cat := range []string{"fiction", "nonfiction"} {
			q := fmt.Sprintf("New York Times bestseller %s %d", cat, year)
			ins, skip := seedByQuery(bookRepo, genreRepo, q, 150)
			totalInserted += ins
			totalSkipped += skip
		}
		logProgress(totalInserted, TARGET_BOOKS)
	}

	// --- PHASE 3: Popular Authors (backbone of catalog) ---
	log.Println("\n========== PHASE 3: Popular Authors ==========")
	for _, a := range popularAuthors {
		ins, skip := seedByAuthor(bookRepo, genreRepo, a.Name, a.Count)
		totalInserted += ins
		totalSkipped += skip
		if totalInserted%500 == 0 {
			logProgress(totalInserted, TARGET_BOOKS)
		}
	}
	logProgress(totalInserted, TARGET_BOOKS)

	// --- PHASE 4: Subject Queries ---
	log.Println("\n========== PHASE 4: Subject Queries ==========")
	for _, q := range subjectQueries {
		ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
		totalInserted += ins
		totalSkipped += skip
		if totalInserted%500 == 0 {
			logProgress(totalInserted, TARGET_BOOKS)
		}
	}
	logProgress(totalInserted, TARGET_BOOKS)

	// --- PHASE 5: Popular Series ---
	log.Println("\n========== PHASE 5: Popular Series ==========")
	for _, q := range seriesQueries {
		ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
		totalInserted += ins
		totalSkipped += skip
	}
	logProgress(totalInserted, TARGET_BOOKS)

	// --- PHASE 6: Trending / BookTok / Recent ---
	log.Println("\n========== PHASE 6: Trending & Recent ==========")
	for _, q := range trendingQueries {
		ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
		totalInserted += ins
		totalSkipped += skip
	}
	logProgress(totalInserted, TARGET_BOOKS)

	// --- PHASE 7: Classics ---
	log.Println("\n========== PHASE 7: Classics ==========")
	for _, q := range classicQueries {
		ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
		totalInserted += ins
		totalSkipped += skip
	}
	logProgress(totalInserted, TARGET_BOOKS)

	// --- PHASE 8: Quality Fill (if still below target) ---
	remaining := TARGET_BOOKS - totalInserted
	if remaining > 0 {
		log.Printf("\n========== PHASE 8: Quality Fill (%d remaining) ==========", remaining)
		fillQueries := []queryConfig{
			{"highly rated fiction 2024", remaining / 12},
			{"highly rated fiction 2023", remaining / 12},
			{"highly rated fiction 2022", remaining / 12},
			{"highly rated nonfiction 2024", remaining / 12},
			{"highly rated nonfiction 2023", remaining / 12},
			{"highly rated nonfiction 2022", remaining / 12},
			{"award winning novels", remaining / 12},
			{"critically acclaimed books", remaining / 12},
			{"best books decade", remaining / 12},
			{"recommended reading list", remaining / 12},
			{"must read books before you die", remaining / 12},
			{"most popular books all time", remaining / 12},
		}
		for _, q := range fillQueries {
			if totalInserted >= TARGET_BOOKS {
				break
			}
			ins, skip := seedByQuery(bookRepo, genreRepo, q.Query, q.Count)
			totalInserted += ins
			totalSkipped += skip
		}
	}

	// --- DONE ---
	log.Println("\n============================================")
	log.Printf("SEEDING COMPLETE")
	log.Printf("Total inserted: %d", totalInserted)
	log.Printf("Total skipped:  %d", totalSkipped)
	log.Printf("Total in DB (approx): %d", len(insertedTitles))
	log.Println("============================================")
}

// ============================================================
// DATABASE CONNECTION
// ============================================================

// connectDB checks DATABASE_URL env var first (for production via flyctl proxy),
// then falls back to local dev config.
//
// Production usage:
//   Terminal 1: flyctl proxy 5432:5432 -a bookmarkd-db
//   Terminal 2: DATABASE_URL="postgres://postgres:PASSWORD@localhost:5432/bookmarkd?sslmode=disable" go run cmd/seed/main.go
//
// Local usage:
//   go run cmd/seed/main.go
func connectDB() *sql.DB {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		log.Println("Using DATABASE_URL from environment (production mode)")
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatal("Failed to connect via DATABASE_URL:", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal("Failed to ping database:", err)
		}
		return db
	}

	log.Println("Using local dev database config")
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
	return db
}

// ============================================================
// CORE SEEDING LOGIC
// ============================================================

func seedByAuthor(repo *database.BookRepository, genreRepo *database.GenreRepository, author string, maxBooks int) (int, int) {
	log.Printf("  Author: %s (target: %d)", author, maxBooks)
	return fetchAndInsert(repo, genreRepo, author, maxBooks, "inauthor")
}

func seedByQuery(repo *database.BookRepository, genreRepo *database.GenreRepository, query string, maxBooks int) (int, int) {
	log.Printf("  Query: %s (target: %d)", query, maxBooks)
	return fetchAndInsert(repo, genreRepo, query, maxBooks, "")
}

func fetchAndInsert(repo *database.BookRepository, genreRepo *database.GenreRepository, searchTerm string, maxBooks int, searchType string) (int, int) {
	inserted := 0
	skipped := 0
	startIndex := 0
	consecutiveEmpty := 0
	maxStartIndex := 1000 // Google Books caps pagination at ~1000

	for inserted < maxBooks && startIndex < maxStartIndex {
		var queryParam string
		if searchType != "" {
			queryParam = fmt.Sprintf("%s:%s", searchType, url.QueryEscape(searchTerm))
		} else {
			queryParam = url.QueryEscape(searchTerm)
		}

		requestURL := fmt.Sprintf("%s?q=%s&startIndex=%d&maxResults=%d&key=%s&orderBy=relevance&langRestrict=en",
			GOOGLE_BOOKS_API, queryParam, startIndex, MAX_RESULTS, getAPIKey())

		books, err := fetchWithRetry(requestURL)
		if err != nil {
			log.Printf("    Error (giving up): %v", err)
			break
		}

		if len(books) == 0 {
			consecutiveEmpty++
			if consecutiveEmpty >= 2 {
				break // No more results for this query
			}
			startIndex += MAX_RESULTS
			continue
		}
		consecutiveEmpty = 0

		for _, book := range books {
			if inserted >= maxBooks {
				break
			}

			if !isQualityBook(book.VolumeInfo) {
				skipped++
				continue
			}

			isbn := extractISBN13(book.VolumeInfo.IndustryIdentifiers)
			year := extractYear(book.VolumeInfo.PublishedDate)
			author := getFirstAuthor(book.VolumeInfo.Authors)

			// Dedup: ISBN
			if isbn != "" && insertedISBNs[isbn] {
				skipped++
				continue
			}

			// Dedup: normalized title+author
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

			// Track for dedup
			if isbn != "" {
				insertedISBNs[isbn] = true
			}
			insertedTitles[bookKey] = createdBook.ID

			// Assign genres from Google Books categories
			assignGenres(genreRepo, createdBook.ID, book.VolumeInfo.Categories)

			inserted++
		}

		startIndex += MAX_RESULTS

		// Base delay between requests
		time.Sleep(time.Duration(BASE_DELAY_MS) * time.Millisecond)
	}

	if inserted > 0 {
		log.Printf("    Done: +%d inserted, %d skipped", inserted, skipped)
	}
	return inserted, skipped
}

// ============================================================
// HTTP WITH RETRY + EXPONENTIAL BACKOFF
// ============================================================

func fetchWithRetry(requestURL string) ([]BookItem, error) {
	backoff := BACKOFF_INITIAL_S

	for attempt := 0; attempt <= MAX_RETRIES; attempt++ {
		resp, err := http.Get(requestURL)
		if err != nil {
			return nil, fmt.Errorf("HTTP error: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var booksResp GoogleBooksResponse
			err := json.NewDecoder(resp.Body).Decode(&booksResp)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("JSON decode error: %w", err)
			}
			return booksResp.Items, nil
		}

		resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode == 503 {
			if attempt == MAX_RETRIES {
				return nil, fmt.Errorf("rate limited after %d retries", MAX_RETRIES)
			}
			// Add jitter to avoid thundering herd
			jitter := rand.Intn(5)
			waitTime := backoff + jitter
			log.Printf("    Rate limited (HTTP %d). Waiting %ds (attempt %d/%d)...",
				resp.StatusCode, waitTime, attempt+1, MAX_RETRIES)
			time.Sleep(time.Duration(waitTime) * time.Second)
			backoff = min(backoff*2, BACKOFF_MAX_S)
			continue
		}

		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("exhausted retries")
}

// ============================================================
// QUALITY FILTERS
// ============================================================

func isQualityBook(info VolumeInfo) bool {
	title := strings.ToLower(info.Title)
	author := strings.ToLower(getFirstAuthor(info.Authors))

	// Must have description
	if len(strings.TrimSpace(info.Description)) < MIN_DESCRIPTION_LEN {
		return false
	}

	// Must have cover image
	if info.ImageLinks.Thumbnail == "" && info.ImageLinks.SmallThumb == "" {
		return false
	}

	// Must have ISBN
	isbn := extractISBN13(info.IndustryIdentifiers)
	if isbn == "" {
		return false
	}

	// Must be English (langRestrict=en in URL handles most, this is a backup)
	if info.Language != "" && info.Language != "en" {
		return false
	}

	// Filter garbage by title keywords
	badKeywords := []string{
		"catalogue", "catalog", "index", "proceedings", "annual report",
		"bibliography", "reference guide", "directory", "handbook of",
		"journal of", "transactions", "bulletin", "circular",
		"workbook", "study guide", "teacher edition", "student edition",
		"test prep", "exam prep", "textbook", "course book",
		"coloring book", "activity book", "puzzle book",
		"for dummies", "complete idiot",
		"summary of", "analysis of", "study guide for",
		"unofficial guide", "unauthorized",
	}

	for _, bad := range badKeywords {
		if strings.Contains(title, bad) {
			return false
		}
	}

	// Filter unknown/invalid authors
	badAuthors := []string{
		"unknown author", "various", "anonymous", "n/a", "none",
		"not available", "editorial", "staff",
	}
	for _, bad := range badAuthors {
		if author == bad || author == "" {
			return false
		}
	}

	// Year range
	year := extractYear(info.PublishedDate)
	if year > 0 && (year < MIN_YEAR || year > MAX_YEAR) {
		return false
	}

	return true
}

// ============================================================
// GENRE ASSIGNMENT
// ============================================================

func assignGenres(genreRepo *database.GenreRepository, bookID int, categories []string) {
	if len(categories) == 0 {
		return
	}

	assignedGenres := make(map[string]bool)

	for _, category := range categories {
		catLower := strings.ToLower(category)

		for genreName, keywords := range genreMapping {
			if assignedGenres[genreName] {
				continue
			}

			for _, keyword := range keywords {
				if strings.Contains(catLower, keyword) {
					genre, err := genreRepo.GetByName(genreName)
					if err != nil || genre == nil {
						continue
					}
					genreRepo.AddGenreToBook(bookID, genre.ID)
					assignedGenres[genreName] = true
					break
				}
			}
		}
	}
}

// ============================================================
// HELPERS
// ============================================================

func loadExistingBooks(db *sql.DB) {
	// Load ISBNs
	rows, err := db.Query("SELECT isbn FROM books WHERE isbn IS NOT NULL AND isbn != ''")
	if err != nil {
		log.Printf("Warning: Could not load existing ISBNs: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var isbn string
			if err := rows.Scan(&isbn); err == nil {
				insertedISBNs[isbn] = true
			}
		}
	}

	// Load title+author combos
	rows2, err := db.Query("SELECT id, title, author FROM books")
	if err != nil {
		log.Printf("Warning: Could not load existing books: %v", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var id int
		var title, author string
		if err := rows2.Scan(&id, &title, &author); err == nil {
			key := normalizeBookKey(title, author)
			insertedTitles[key] = id
		}
	}
}

func normalizeBookKey(title, author string) string {
	title = strings.ToLower(strings.TrimSpace(title))

	// Remove leading articles
	title = regexp.MustCompile(`^(the|a|an)\s+`).ReplaceAllString(title, "")

	// Remove subtitle (after : or " - ")
	if idx := strings.Index(title, ":"); idx > 0 {
		title = title[:idx]
	}
	if idx := strings.Index(title, " - "); idx > 0 {
		title = title[:idx]
	}

	// Remove punctuation and collapse whitespace
	title = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(title, "")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Normalize author
	author = strings.ToLower(strings.TrimSpace(author))
	author = regexp.MustCompile(`\s+`).ReplaceAllString(author, " ")

	return title + "|" + author
}

func extractISBN13(identifiers []IndustryIdentifier) string {
	// Prefer ISBN-13
	for _, id := range identifiers {
		if id.Type == "ISBN_13" {
			return id.Identifier
		}
	}
	// Fall back to ISBN-10
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

func logProgress(current, target int) {
	pct := float64(current) / float64(target) * 100
	log.Printf(">>> PROGRESS: %d / %d (%.1f%%)", current, target, pct)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}