package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pulkyeet/BookmarkD/internal/database"
	"github.com/pulkyeet/BookmarkD/internal/middleware"
	"github.com/pulkyeet/BookmarkD/internal/models"
)

type UserHandler struct {
	userRepo   *database.UserRepository
	followRepo *database.FollowRepository
}

func NewUserHandler(userRepo *database.UserRepository, followRepo *database.FollowRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo, followRepo: followRepo}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get viewer ID if authenticated (optional auth)
	var viewerID *int
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			if claims, err := models.ValidateToken(parts[1]); err == nil {
				viewerID = &claims.UserID
			}
		}
	}

	profile, err := h.userRepo.GetProfile(userID, viewerID)
	if err != nil {
		log.Printf("GetProfile error for user %d: %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *UserHandler) Follow(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	followingID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	if claims.UserID == followingID {
		http.Error(w, "You can't follow yourself!", http.StatusForbidden)
		return
	}
	err = h.followRepo.Follow(claims.UserID, followingID)
	if err != nil {
		http.Error(w, "Failed to follow user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Follow successful!"})
}

func (h *UserHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	followingID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	err = h.followRepo.Unfollow(claims.UserID, followingID)
	if err != nil {
		http.Error(w, "Failed to unfollow user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Unfollow successful!"})
}

func (h *UserHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	followers, err := h.followRepo.GetFollowers(userID)
	if err != nil {
		http.Error(w, "Failed to get followers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(followers)
}

func (h *UserHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	following, err := h.followRepo.GetFollowing(userID)
	if err != nil {
		http.Error(w, "Failed to get followers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(following)
}

type UserHandlerWithStats struct {
	*UserHandler
	ratingRepo *database.RatingRepository
}

func NewUserHandlerWithStats(userRepo *database.UserRepository, followRepo *database.FollowRepository, ratingRepo *database.RatingRepository) *UserHandlerWithStats {
	return &UserHandlerWithStats{
		UserHandler: NewUserHandler(userRepo, followRepo),
		ratingRepo:  ratingRepo,
	}
}

func (h *UserHandlerWithStats) GetYearStats(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(parts[5])
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}
	stats, err := h.ratingRepo.GetYearStats(userID, year)
	if err != nil {
		log.Printf("GetYearStats error: %v", err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
