package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/pulkyeet/bookrate/internal/database"
	"github.com/pulkyeet/bookrate/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	userRepo    *database.UserRepository
	oauthConfig *oauth2.Config
}

type FinaliseOAuthRequest struct {
	PendingToken string `json:"pending_token"`
	Username     string `json:"username"`
}

func NewAuthHandler(userRepo *database.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

func (h *AuthHandler) SetOAuthConfig(clientID, clientSecret, redirectURL string) {
	h.oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
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

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state, err := generateStateToken()
	if err != nil {
		http.Error(w, "failed to generate state token", http.StatusInternalServerError)
		return
	}
	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	client := h.oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	log.Printf("Google OAuth: email=%s, google_id=%s", googleUser.Email, googleUser.ID)

	// Step 1: Check if user exists by Google ID
	user, err := h.userRepo.GetByGoogleID(googleUser.ID)
	if err == nil {
		// Found by google_id - existing Google user
		log.Printf("Found existing user by google_id: user_id=%d", user.ID)
		jwtToken, err := models.GenerateToken(user)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		redirectURL := fmt.Sprintf("http://localhost:8080/auth-success.html?token=%s", jwtToken)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	log.Printf("User not found by google_id, error: %v", err)

	// Step 2: Check if user exists by email
	existingUser, err := h.userRepo.GetByEmail(googleUser.Email)
	if err == nil {
		// Email exists - check if Google is already linked
		if existingUser.GoogleID != nil {
			// Google already linked but GetByGoogleID failed - data inconsistency
			log.Printf("WARNING: Email exists with google_id but GetByGoogleID failed. Logging in anyway. user_id=%d", existingUser.ID)
			jwtToken, err := models.GenerateToken(existingUser)
			if err != nil {
				http.Error(w, "Failed to generate token", http.StatusInternalServerError)
				return
			}

			redirectURL := fmt.Sprintf("http://localhost:8080/auth-success.html?token=%s", jwtToken)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}

		// Email exists but no Google linked - link it now
		log.Printf("Linking Google account to existing user_id=%d", existingUser.ID)
		err = h.userRepo.LinkGoogleAccount(existingUser.ID, googleUser.ID)
		if err != nil {
			log.Printf("Failed to link Google account: %v", err)
			http.Error(w, "Failed to link account", http.StatusInternalServerError)
			return
		}

		// Refresh user data
		existingUser.GoogleID = &googleUser.ID

		jwtToken, err := models.GenerateToken(existingUser)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		redirectURL := fmt.Sprintf("http://localhost:8080/auth-success.html?token=%s", jwtToken)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	log.Printf("User not found by email either, creating new user")

	// Step 3: Completely new user - show username picker
	pendingToken, err := models.GeneratePendingOAuthToken(googleUser.Email, googleUser.ID, googleUser.Name)
	if err != nil {
		http.Error(w, "Failed to generate pending token", http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("http://localhost:8080/choose-username.html?pending=%s&email=%s", pendingToken, googleUser.Email)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) FinaliseOAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FinaliseOAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	claims, err := models.ValidatePendingOAuthToken(req.PendingToken)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}
	if req.Username == "" || len(req.Username) < 3 {
		http.Error(w, "Username must 3 or more characters", http.StatusForbidden)
		return
	}
	user, err := h.userRepo.CreateWithGoogle(claims.Email, req.Username, claims.GoogleID)
	if err != nil {
		if err == models.ErrUsernameExists {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	jwtToken, err := models.GenerateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	response := LoginResponse{
		Token: jwtToken,
		User:  user,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
