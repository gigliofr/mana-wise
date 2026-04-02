package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
)

type authPlanMockUserRepo struct {
	user *domain.User
}

func (r *authPlanMockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if r.user != nil && r.user.ID == id {
		copyUser := *r.user
		return &copyUser, nil
	}
	return nil, nil
}

func (r *authPlanMockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.user != nil && strings.EqualFold(r.user.Email, email) {
		copyUser := *r.user
		return &copyUser, nil
	}
	return nil, nil
}

func (r *authPlanMockUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.user = user
	return nil
}

func (r *authPlanMockUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.user = user
	return nil
}

func (r *authPlanMockUserRepo) CheckAndIncrementDailyAnalyses(ctx context.Context, userID, today string, limit int) (bool, error) {
	return true, nil
}

func TestUpdatePlan_DowngradeBlockedWhenProEntitlementActive(t *testing.T) {
	expires := time.Now().UTC().Add(15 * 24 * time.Hour)
	repo := &authPlanMockUserRepo{user: &domain.User{
		ID:        "u-1",
		Email:     "u1@example.com",
		Name:      "U1",
		Plan:      domain.PlanPro,
		ProUntil:  &expires,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}}

	h := NewAuthHandler(repo, "secret", 24)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/plan", strings.NewReader(`{"plan":"free"}`))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.UpdatePlan(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.user.Plan != domain.PlanPro {
		t.Fatalf("expected persisted plan to remain pro, got %s", repo.user.Plan)
	}
}

func TestUpdatePlan_DowngradeAllowedWhenNoActiveEntitlement(t *testing.T) {
	repo := &authPlanMockUserRepo{user: &domain.User{
		ID:        "u-2",
		Email:     "u2@example.com",
		Name:      "U2",
		Plan:      domain.PlanPro,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}}

	h := NewAuthHandler(repo, "secret", 24)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/plan", strings.NewReader(`{"plan":"free"}`))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-2"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.UpdatePlan(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out TokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.User == nil || out.User.Plan != domain.PlanFree {
		t.Fatalf("expected response user free, got %#v", out.User)
	}
	if repo.user.Plan != domain.PlanFree {
		t.Fatalf("expected persisted plan free, got %s", repo.user.Plan)
	}
}
