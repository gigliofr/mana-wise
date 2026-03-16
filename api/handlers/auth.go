package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles user registration and login.
type AuthHandler struct {
	userRepo    domain.UserRepository
	jwtSecret   string
	expiryHours int
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(userRepo domain.UserRepository, jwtSecret string, expiryHours int) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSecret: jwtSecret, expiryHours: expiryHours}
}

// RegisterRequest is the JSON body for POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest is the JSON body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is returned by register and login.
type TokenResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		jsonError(w, "email, password and name are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	existing, err := h.userRepo.FindByEmail(r.Context(), req.Email)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		jsonError(w, "email already registered", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		Plan:         domain.PlanFree,
		CreatedAt:    time.Now().UTC(),
	}
	if err = h.userRepo.Create(r.Context(), user); err != nil {
		jsonError(w, "could not create user", http.StatusInternalServerError)
		return
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())

	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan), h.jwtSecret, h.expiryHours)
	if err != nil {
		jsonError(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TokenResponse{Token: token, User: user})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.userRepo.FindByEmail(r.Context(), req.Email)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan), h.jwtSecret, h.expiryHours)
	if err != nil {
		jsonError(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())

	jsonOK(w, TokenResponse{Token: token, User: user})
}

// Me handles GET /auth/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())
	jsonOK(w, user)
}

// Health handles GET /api/v1/health.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}
