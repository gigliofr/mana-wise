package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"golang.org/x/crypto/bcrypt"
)

type authResetMockUserRepo struct {
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
}

func (r *authResetMockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if u, ok := r.byID[id]; ok {
		copyUser := *u
		return &copyUser, nil
	}
	return nil, nil
}

func (r *authResetMockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := r.byEmail[strings.ToLower(strings.TrimSpace(email))]; ok {
		copyUser := *u
		return &copyUser, nil
	}
	return nil, nil
}

func (r *authResetMockUserRepo) Create(ctx context.Context, user *domain.User) error {
	if r.byID == nil {
		r.byID = map[string]*domain.User{}
	}
	if r.byEmail == nil {
		r.byEmail = map[string]*domain.User{}
	}
	r.byID[user.ID] = user
	r.byEmail[strings.ToLower(strings.TrimSpace(user.Email))] = user
	return nil
}

func (r *authResetMockUserRepo) Update(ctx context.Context, user *domain.User) error {
	if r.byID == nil {
		r.byID = map[string]*domain.User{}
	}
	if r.byEmail == nil {
		r.byEmail = map[string]*domain.User{}
	}
	r.byID[user.ID] = user
	r.byEmail[strings.ToLower(strings.TrimSpace(user.Email))] = user
	return nil
}

func (r *authResetMockUserRepo) CheckAndIncrementDailyAnalyses(ctx context.Context, userID, today string, limit int) (bool, error) {
	return true, nil
}

type authResetMockTokenRepo struct {
	tokens map[string]*domain.PasswordResetToken
}

func (r *authResetMockTokenRepo) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	if r.tokens == nil {
		r.tokens = map[string]*domain.PasswordResetToken{}
	}
	r.tokens[token.Token] = token
	return nil
}

func (r *authResetMockTokenRepo) Consume(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	if r.tokens == nil {
		return nil, nil
	}
	rec, ok := r.tokens[token]
	if !ok {
		return nil, nil
	}
	if !rec.ExpiresAt.After(time.Now().UTC()) {
		delete(r.tokens, token)
		return nil, nil
	}
	delete(r.tokens, token)
	copyRec := *rec
	return &copyRec, nil
}

type sentEmail struct {
	to      string
	subject string
	text    string
	html    string
}

type authResetMockMailer struct {
	sent []sentEmail
}

func (m *authResetMockMailer) Send(to, subject, textBody, htmlBody string) error {
	m.sent = append(m.sent, sentEmail{to: to, subject: subject, text: textBody, html: htmlBody})
	return nil
}

func TestForgotPassword_CreatesTokenAndSendsEmail(t *testing.T) {
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-1": {
				ID:           "u-1",
				Email:        "user@example.com",
				PasswordHash: "hash",
				Name:         "User",
				Plan:         domain.PlanFree,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	repo.byEmail["user@example.com"] = repo.byID["u-1"]

	tokenRepo := &authResetMockTokenRepo{}
	mailer := &authResetMockMailer{}
	h := NewAuthHandler(repo, "secret", 24).WithPasswordResetRepo(tokenRepo).WithMailer(mailer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(`{"email":"user@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ForgotPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(tokenRepo.tokens) != 1 {
		t.Fatalf("expected one reset token, got %d", len(tokenRepo.tokens))
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one email sent, got %d", len(mailer.sent))
	}
	if mailer.sent[0].to != "user@example.com" {
		t.Fatalf("expected recipient user@example.com, got %s", mailer.sent[0].to)
	}
}

func TestRegister_CreatesPendingAccountAndSendsVerificationEmail(t *testing.T) {
	repo := &authResetMockUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
	tokenRepo := &authResetMockTokenRepo{}
	mailer := &authResetMockMailer{}
	h := NewAuthHandler(repo, "secret", 24).WithPasswordResetRepo(tokenRepo).WithMailer(mailer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"welcome@example.com","password":"StrongPass123","name":"Welcome User"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload RegisterResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if !payload.RequiresVerification {
		t.Fatalf("expected requires_verification=true")
	}

	if len(tokenRepo.tokens) != 1 {
		t.Fatalf("expected one verification token, got %d", len(tokenRepo.tokens))
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one verification email, got %d", len(mailer.sent))
	}
	if mailer.sent[0].to != "welcome@example.com" {
		t.Fatalf("unexpected recipient: %s", mailer.sent[0].to)
	}
	if !strings.Contains(strings.ToLower(mailer.sent[0].subject), "verify") {
		t.Fatalf("expected verification email subject, got %q", mailer.sent[0].subject)
	}

	created, _ := repo.FindByEmail(context.Background(), "welcome@example.com")
	if created == nil {
		t.Fatalf("expected created user")
	}
	if !created.EmailVerificationPending {
		t.Fatalf("expected email verification pending=true")
	}
}

func TestLogin_UnverifiedEmail_ReturnsForbidden(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("StrongPass123"), bcrypt.DefaultCost)
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-login": {
				ID:                       "u-login",
				Email:                    "login@example.com",
				PasswordHash:             string(hash),
				Name:                     "Login User",
				Plan:                     domain.PlanFree,
				EmailVerificationPending: true,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	repo.byEmail["login@example.com"] = repo.byID["u-login"]
	h := NewAuthHandler(repo, "secret", 24)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"login@example.com","password":"StrongPass123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestVerifyEmail_ConsumesTokenAndActivatesAccount(t *testing.T) {
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-verify": {
				ID:                       "u-verify",
				Email:                    "verify@example.com",
				PasswordHash:             "hash",
				Name:                     "Verify User",
				Plan:                     domain.PlanFree,
				EmailVerificationPending: true,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	repo.byEmail["verify@example.com"] = repo.byID["u-verify"]
	tokenRepo := &authResetMockTokenRepo{tokens: map[string]*domain.PasswordResetToken{
		"verify-token": {
			Token:     "verify-token",
			UserID:    "u-verify",
			Email:     "verify@example.com",
			Purpose:   "email_verification",
			ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
	}}

	h := NewAuthHandler(repo, "secret", 24).WithPasswordResetRepo(tokenRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=verify-token", nil)
	rr := httptest.NewRecorder()

	h.VerifyEmail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	updated, _ := repo.FindByID(context.Background(), "u-verify")
	if updated == nil {
		t.Fatalf("expected updated user")
	}
	if updated.EmailVerificationPending {
		t.Fatalf("expected email verification pending=false")
	}
	if updated.EmailVerifiedAt == nil {
		t.Fatalf("expected EmailVerifiedAt to be set")
	}
}

func TestResetPassword_ConsumesTokenAndUpdatesPassword(t *testing.T) {
	oldHash, _ := bcrypt.GenerateFromPassword([]byte("OldPass123"), bcrypt.DefaultCost)
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-2": {
				ID:           "u-2",
				Email:        "reset@example.com",
				PasswordHash: string(oldHash),
				Name:         "Reset",
				Plan:         domain.PlanFree,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	repo.byEmail["reset@example.com"] = repo.byID["u-2"]

	tokenRepo := &authResetMockTokenRepo{tokens: map[string]*domain.PasswordResetToken{
		"tok-123": {
			Token:     "tok-123",
			UserID:    "u-2",
			Email:     "reset@example.com",
			Purpose:   "password_reset",
			ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
	}}
	mailer := &authResetMockMailer{}
	h := NewAuthHandler(repo, "secret", 24).WithPasswordResetRepo(tokenRepo).WithMailer(mailer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(`{"token":"tok-123","new_password":"NewPass456"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ResetPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	updated, _ := repo.FindByID(context.Background(), "u-2")
	if updated == nil {
		t.Fatalf("expected updated user")
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("NewPass456")) != nil {
		t.Fatalf("password hash was not updated")
	}
	if _, ok := tokenRepo.tokens["tok-123"]; ok {
		t.Fatalf("token should be consumed and removed")
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one confirmation email, got %d", len(mailer.sent))
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not valid json: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", payload["status"])
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	repo := &authResetMockUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
	tokenRepo := &authResetMockTokenRepo{tokens: map[string]*domain.PasswordResetToken{}}
	h := NewAuthHandler(repo, "secret", 24).WithPasswordResetRepo(tokenRepo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(`{"token":"missing","new_password":"NewPass456"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ResetPassword(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRefresh_ReturnsNewTokenForAuthenticatedUser(t *testing.T) {
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-refresh": {
				ID:           "u-refresh",
				Email:        "refresh@example.com",
				PasswordHash: "hash",
				Name:         "Refresh User",
				Plan:         domain.PlanFree,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	h := NewAuthHandler(repo, "secret", 24)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-refresh")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload TokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if strings.TrimSpace(payload.Token) == "" {
		t.Fatalf("expected refreshed token in response")
	}
	if strings.TrimSpace(payload.RefreshToken) == "" {
		t.Fatalf("expected refreshed refresh_token in response")
	}
	if payload.User == nil || payload.User.ID != "u-refresh" {
		t.Fatalf("expected refreshed user payload")
	}
}

func TestRefresh_WithRefreshTokenBody_ReturnsNewTokenPair(t *testing.T) {
	repo := &authResetMockUserRepo{
		byID: map[string]*domain.User{
			"u-refresh": {
				ID:           "u-refresh",
				Email:        "refresh@example.com",
				PasswordHash: "hash",
				Name:         "Refresh User",
				Plan:         domain.PlanFree,
			},
		},
		byEmail: map[string]*domain.User{},
	}
	h := NewAuthHandler(repo, "secret", 5, 60)

	refreshToken, err := middleware.GenerateRefreshToken("u-refresh", "refresh@example.com", string(domain.PlanFree), "secret", 60)
	if err != nil {
		t.Fatalf("expected refresh token generation: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload TokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if strings.TrimSpace(payload.Token) == "" {
		t.Fatalf("expected refreshed access token in response")
	}
	if strings.TrimSpace(payload.RefreshToken) == "" {
		t.Fatalf("expected refreshed refresh token in response")
	}
}

func TestRefresh_Unauthenticated(t *testing.T) {
	repo := &authResetMockUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
	h := NewAuthHandler(repo, "secret", 24)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
