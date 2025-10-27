package handlers

import (
	"encoding/json"
	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/models"
	"log"
	"net/http"
)

type AuthHandler struct {
	userRepo *database.UserRepository
}

func NewAuthHandler(userRepo *database.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

// Signup types and handler
type SignupRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	passwordHash, err := models.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.Create(req.Email, req.Username, passwordHash)
	if err != nil {
		if err == models.ErrEmailExists {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		if err == models.ErrUsernameExists {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Login types and handler (ADD BELOW)
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		log.Printf("GetByEmail error: %v", err)
		if err == models.ErrInvalidCredentials {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := models.CheckPassword(req.Password, user.PasswordHash); err != nil {
		log.Printf("CheckPassword error: %v", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := models.GenerateToken(user)
	if err != nil {
		log.Printf("GenerateToken error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
