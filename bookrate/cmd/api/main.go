package main

import (
	"encoding/json"
	"fmt"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/handlers"
	"github.com/pulkyeet/bookrate/internal/middleware"
	"log"
	"net/http"
)

func main() {
	dbConfig := database.Config{
		Host:     "localhost",
		Port:     5433,
		User:     "bookrate",
		Password: "localdev2178",
		DBName:   "bookrate",
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialising repositories and handlers
	userRepo := database.NewUserRepository(db)
	bookRepo := database.NewBookRepository(db)
	ratingRepo := database.NewRatingRepository(db)
	followRepo := database.NewFollowRepository(db)
	commentRepo := database.NewCommentRepository(db)
	genreRepo := database.NewGenreRepository(db)
	listRepo := database.NewListRepository(db)

	ratingHandler := handlers.NewRatingHandler(ratingRepo)
	bookHandler := handlers.NewBookHandler(bookRepo)
	authHandler := handlers.NewAuthHandler(userRepo)
	feedHandler := handlers.NewFeedHandler(ratingRepo)
	userHandler := handlers.NewUserHandler(userRepo, followRepo)
	commentHandler := handlers.NewCommentHandler(commentRepo)
	genreHandler := handlers.NewGenreHandler(genreRepo)
	listHandler := handlers.NewListHandler(listRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/auth/signup", authHandler.Signup)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	// Protected route
	mux.HandleFunc("/api/profile", middleware.AuthMiddleware(profileHandler))
	mux.HandleFunc("/api/books", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			bookHandler.List(w, r)
		} else if r.Method == http.MethodPost {
			middleware.AuthMiddleware(bookHandler.Create)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/books/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			bookHandler.Get(w, r)
		} else if r.Method == http.MethodPut || r.Method == http.MethodPatch {
			middleware.AuthMiddleware(bookHandler.Update)(w, r)
		} else if r.Method == http.MethodDelete {
			middleware.AuthMiddleware(bookHandler.Delete)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/books/{id}/ratings/me", func(w http.ResponseWriter, r *http.Request) {
		bookID := r.PathValue("id")
		query := r.URL.Query()
		query.Set("book_id", bookID)
		r.URL.RawQuery = query.Encode()
		if r.Method == http.MethodGet {
			middleware.AuthMiddleware(ratingHandler.GetMyRatingForBook)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/books/{id}/ratings", func(w http.ResponseWriter, r *http.Request) {
		bookID := r.PathValue("id")
		query := r.URL.Query()
		query.Set("book_id", bookID)
		r.URL.RawQuery = query.Encode()

		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(ratingHandler.CreateRating)(w, r)
		case http.MethodGet:
			middleware.OptionalAuthMiddleware(ratingHandler.GetBookRatings)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(ratingHandler.DeleteRating)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/users/me/ratings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			middleware.AuthMiddleware(ratingHandler.GetMyRatings)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/ratings/{id}/like", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(ratingHandler.LikeRating)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(ratingHandler.UnlikeRating)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/users/{id}/profile", userHandler.GetProfile)
	mux.HandleFunc("/api/users/{id}/follow", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(userHandler.Follow)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(userHandler.Unfollow)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/ratings/{id}/comments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(commentHandler.CreateComment)(w, r)
		case http.MethodGet:
			commentHandler.GetComments(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/comments/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			middleware.AuthMiddleware(commentHandler.DeleteComment)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/ratings/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			middleware.AuthMiddleware(ratingHandler.UpdateRating)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/users/{id}/followers", userHandler.GetFollowers)
	mux.HandleFunc("/api/users/{id}/following", userHandler.GetFollowing)
	mux.HandleFunc("/api/feed", middleware.OptionalAuthMiddleware(feedHandler.GetFeed))
	mux.HandleFunc("/api/books/trending", bookHandler.GetTrending)
	mux.HandleFunc("/api/books/popular", bookHandler.GetPopular)
	mux.HandleFunc("/api/books/{id}/similar", bookHandler.GetSimilar)
	mux.HandleFunc("/api/lists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(listHandler.Create)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/lists/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listHandler.GetByID(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(listHandler.Update)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(listHandler.Delete)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/lists/{id}/books", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(listHandler.AddBook)(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(listHandler.ReorderBooks)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/lists/{id}/books/{bookID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			middleware.AuthMiddleware(listHandler.RemoveBook)(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/lists/{id}/bookmark", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.AuthMiddleware(listHandler.BookmarkList)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(listHandler.UnbookmarkList)(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/users/me/bookmarked-lists", middleware.AuthMiddleware(listHandler.GetBookmarkedLists))
	mux.HandleFunc("/api/lists/popular", listHandler.GetPopularLists)
	mux.HandleFunc("/api/users/{id}/lists", listHandler.GetUserLists)

	mux.HandleFunc("/api/genres", genreHandler.GetAll)

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	port := "8080"
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	// Wrap mux with CORS middleware
	handler := middleware.CORS(mux)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("BookRate API"))
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Profile Data",
		"user_id":  claims.UserID,
		"email":    claims.Email,
		"username": claims.Username,
	})
}
