package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles user registration and login.
type AuthHandler struct {
	userRepo      domain.UserRepository
	resetTokenRepo domain.PasswordResetTokenRepository
	mailer        domain.EmailSender
	jwtSecret     string
	expiryHours   int
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(userRepo domain.UserRepository, jwtSecret string, expiryHours int) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		mailer:      domain.NoopEmailSender{},
		jwtSecret:   jwtSecret,
		expiryHours: expiryHours,
	}
}

// WithPasswordResetRepo enables forgot/reset-password token persistence.
func (h *AuthHandler) WithPasswordResetRepo(repo domain.PasswordResetTokenRepository) *AuthHandler {
	h.resetTokenRepo = repo
	return h
}

// WithMailer enables transactional auth emails.
func (h *AuthHandler) WithMailer(mailer domain.EmailSender) *AuthHandler {
	if mailer != nil {
		h.mailer = mailer
	}
	return h
}

// RegisterRequest is the JSON body for POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// UpdatePlanRequest is the JSON body for POST /auth/plan.
type UpdatePlanRequest struct {
	Plan              string `json:"plan"`
	DonationTier      string `json:"donation_tier,omitempty"`
	DonationReference string `json:"donation_reference,omitempty"`
}

// LoginRequest is the JSON body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ForgotPasswordRequest is the JSON body for POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the JSON body for POST /auth/reset-password.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
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

	welcomeText := "Welcome to ManaWise. Your account has been created successfully."
	welcomeHTML := "<p>Welcome to ManaWise.</p><p>Your account has been created successfully.</p>"
	_ = h.sendEmail(user.Email, "Welcome to ManaWise", welcomeText, welcomeHTML)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TokenResponse{Token: token, User: user})
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		jsonError(w, "email is required", http.StatusBadRequest)
		return
	}

	if h.resetTokenRepo == nil {
		jsonOK(w, map[string]any{"status": "accepted", "message": "if the email exists, reset instructions will be sent"})
		return
	}

	user, err := h.userRepo.FindByEmail(r.Context(), email)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user != nil {
		token, err := generateHexToken(24)
		if err == nil {
			ttl := passwordResetTokenTTL()
			expiresAt := time.Now().UTC().Add(ttl)
			rec := &domain.PasswordResetToken{
				Token:     token,
				UserID:    user.ID,
				Email:     user.Email,
				Purpose:   "password_reset",
				ExpiresAt: expiresAt,
				CreatedAt: time.Now().UTC(),
			}
			if err := h.resetTokenRepo.Create(r.Context(), rec); err == nil {
				resetURL := buildActionURL(strings.TrimSpace(os.Getenv("FRONTEND_RESET_PASSWORD_URL")), "/reset-password", token)
				log.Printf("[auth] reset-password link generated user=%s url=%s", user.Email, redactActionURLForLog(resetURL))
				textBody := "We received a password reset request.\nUse this link to reset your password: " + resetURL + "\nExpires at: " + expiresAt.Format(time.RFC3339)
				htmlBody := "<p>We received a password reset request.</p><p>Use this link to reset your password: <a href=\"" + resetURL + "\">reset password</a></p><p>Expires at: " + expiresAt.Format(time.RFC3339) + "</p>"
				_ = h.sendEmail(user.Email, "ManaWise password reset", textBody, htmlBody)
			}
		}
	}

	jsonOK(w, map[string]any{"status": "accepted", "message": "if the email exists, reset instructions will be sent"})
}

// ResetPassword handles POST /auth/reset-password.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(strings.TrimSpace(req.NewPassword)) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		jsonError(w, "token is required", http.StatusBadRequest)
		return
	}
	if h.resetTokenRepo == nil {
		jsonError(w, "password reset is not configured", http.StatusServiceUnavailable)
		return
	}

	rec, err := h.resetTokenRepo.Consume(r.Context(), strings.TrimSpace(req.Token))
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if rec == nil || rec.Purpose != "password_reset" {
		jsonError(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), rec.UserID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(req.NewPassword)), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now().UTC()
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		jsonError(w, "could not update password", http.StatusInternalServerError)
		return
	}

	confirmText := "Your ManaWise password has been changed successfully."
	confirmHTML := "<p>Your ManaWise password has been changed successfully.</p>"
	_ = h.sendEmail(user.Email, "ManaWise password changed", confirmText, confirmHTML)

	jsonOK(w, map[string]any{"status": "ok"})
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

	if user.Plan == domain.PlanPro && user.ProUntil != nil && !user.ProUntil.After(time.Now().UTC()) {
		user.Plan = domain.PlanFree
		user.ProUntil = nil
		_ = h.userRepo.Update(r.Context(), user)
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan), h.jwtSecret, h.expiryHours)
	if err != nil {
		jsonError(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())

	jsonOK(w, TokenResponse{Token: token, User: user})
}

// UpdatePlan handles POST /auth/plan for authenticated users.
func (h *AuthHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonError(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	var req UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	plan := domain.Plan(strings.ToLower(strings.TrimSpace(req.Plan)))
	switch plan {
	case domain.PlanFree, domain.PlanPro:
	default:
		jsonError(w, "invalid plan: supported values are free, pro", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	if user.Plan != plan {
		if plan == domain.PlanFree && user.Plan == domain.PlanPro && user.ProUntil != nil && user.ProUntil.After(time.Now().UTC()) {
			jsonError(w, fmt.Sprintf("active pro entitlement until %s", user.ProUntil.UTC().Format(time.RFC3339)), http.StatusConflict)
			return
		}
		user.Plan = plan
		if plan == domain.PlanFree {
			user.ProUntil = nil
		}
		user.UpdatedAt = time.Now().UTC()
	}

	if plan == domain.PlanPro {
		tier := strings.ToLower(strings.TrimSpace(req.DonationTier))
		ref := strings.TrimSpace(req.DonationReference)
		if ref == "" {
			jsonError(w, "donation_reference is required to activate pro", http.StatusBadRequest)
			return
		}
		base := time.Now().UTC()
		if user.ProUntil != nil && user.ProUntil.After(base) {
			base = *user.ProUntil
		}

		switch tier {
		case "beta_month_1eur":
			expires := base.AddDate(0, 1, 0)
			user.ProUntil = &expires
		case "beta_year_190eur":
			expires := base.AddDate(1, 0, 0)
			user.ProUntil = &expires
		default:
			jsonError(w, "invalid donation_tier: use beta_month_1eur or beta_year_190eur", http.StatusBadRequest)
			return
		}

		user.UpdatedAt = time.Now().UTC()
	}

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		jsonError(w, "could not update plan", http.StatusInternalServerError)
		return
	}

	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())
	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan), h.jwtSecret, h.expiryHours)
	if err != nil {
		jsonError(w, "could not generate token", http.StatusInternalServerError)
		return
	}

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
	if user.Plan == domain.PlanPro && user.ProUntil != nil && !user.ProUntil.After(time.Now().UTC()) {
		user.Plan = domain.PlanFree
		user.ProUntil = nil
		_ = h.userRepo.Update(r.Context(), user)
	}
	user.Remaining = user.RemainingAnalyses(domain.CurrentBusinessDay())
	jsonOK(w, user)
}

func (h *AuthHandler) sendEmail(to, subject, textBody, htmlBody string) error {
	if h.mailer == nil {
		return nil
	}
	return h.mailer.Send(to, subject, textBody, htmlBody)
}

func generateHexToken(size int) (string, error) {
	if size <= 0 {
		size = 24
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func passwordResetTokenTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PASSWORD_RESET_TOKEN_TTL_MINUTES"))
	if raw == "" {
		return 30 * time.Minute
	}
	v, err := time.ParseDuration(raw + "m")
	if err != nil || v <= 0 {
		return 30 * time.Minute
	}
	return v
}

func buildActionURL(base, defaultPath, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}

	base = strings.TrimSpace(base)
	if base == "" {
		base = strings.TrimSpace(defaultPath)
	}
	if base == "" {
		base = "/reset-password"
	}

	if u, err := url.Parse(base); err == nil {
		q := u.Query()
		for key := range q {
			cleanKey := strings.ToLower(strings.TrimSpace(key))
			if strings.Contains(cleanKey, "token") {
				q.Del(key)
			}
		}
		q.Set("token", token)
		u.RawQuery = q.Encode()
		return u.String()
	}

	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	if strings.HasSuffix(base, "?") || strings.HasSuffix(base, "&") {
		sep = ""
	}
	return base + sep + "token=" + url.QueryEscape(token)
}

func redactTokenForLog(token string) string {
	clean := strings.TrimSpace(token)
	if clean == "" {
		return ""
	}
	if len(clean) <= 8 {
		return "***"
	}
	return clean[:4] + "..." + clean[len(clean)-4:]
}

func redactActionURLForLog(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	q := u.Query()
	for key := range q {
		if strings.Contains(strings.ToLower(strings.TrimSpace(key)), "token") {
			vals := q[key]
			for i := range vals {
				vals[i] = redactTokenForLog(vals[i])
			}
			q[key] = vals
		}
	}
	u.RawQuery = q.Encode()

	for _, prefix := range []string{"/verify-email/", "/reset-password/"} {
		pathLower := strings.ToLower(u.Path)
		idx := strings.Index(pathLower, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		if start >= len(u.Path) {
			continue
		}
		tail := u.Path[start:]
		sepIdx := strings.IndexByte(tail, '/')
		tokenPart := tail
		rest := ""
		if sepIdx >= 0 {
			tokenPart = tail[:sepIdx]
			rest = tail[sepIdx:]
		}
		u.Path = u.Path[:start] + redactTokenForLog(tokenPart) + rest
		break
	}

	return u.String()
}

// Health handles GET /api/v1/health.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}
