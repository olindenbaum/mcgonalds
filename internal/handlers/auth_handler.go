package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// SignupRequest represents the expected payload for signup
type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents the expected payload for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Signup handles user registration
// Signup godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body SignupRequest true "User signup information"
// @Success 201 {object} map[string]string "User created successfully"
// @Failure 400 {string} string "Invalid request payload or user creation error"
// @Failure 500 {string} string "Error processing password"
// @Router /signup [post]
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	log.Println("Signup request received")
	var req SignupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error processing password: %v", err)
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	user := model.User{
		Username: req.Username,
		Password: string(hashedPassword),
	}

	if err := h.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Error creating user. Username may already be in use.", http.StatusBadRequest)
		return
	}

	log.Printf("User created successfully: %s", user.Username)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created successfully",
	})
}

// Login godoc
// @Summary Authenticate user
// @Description Authenticate a user and get a JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login information"
// @Success 200 {object} map[string]string "Authentication successful"
// @Failure 401 {string} string "Invalid username or password"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	log.Println("Login request received")
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Printf("Invalid login attempt for username: %s", req.Username)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Compare the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("Invalid password attempt for user: %s", user.Username)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Username)
	if err != nil {
		log.Printf("Error generating token for user %s: %v", user.Username, err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s logged in successfully", user.Username)
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
