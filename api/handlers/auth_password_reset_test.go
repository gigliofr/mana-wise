package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestRegister_SendsWelcomeEmail(t *testing.T) {
	repo := &authResetMockUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
	mailer := &authResetMockMailer{}
	h := NewAuthHandler(repo, "secret", 24).WithMailer(mailer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"welcome@example.com","password":"StrongPass123","name":"Welcome User"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one welcome email, got %d", len(mailer.sent))
	}
	if mailer.sent[0].to != "welcome@example.com" {
		t.Fatalf("unexpected recipient: %s", mailer.sent[0].to)
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
