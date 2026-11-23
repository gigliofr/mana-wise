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
)

func postPlan(t *testing.T, h *AuthHandler, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/plan", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, userID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdatePlan(rr, req)
	return rr
}

// Security regression: a free user must not be able to self-activate Pro by
// sending an arbitrary donation reference.
func TestUpdatePlan_SelfActivationOfProIsRejected(t *testing.T) {
	repo := &authPlanMockUserRepo{user: &domain.User{ID: "u-free", Email: "free@example.com", Plan: domain.PlanFree}}
	h := NewAuthHandler(repo, "secret", 24)

	for _, body := range []string{
		`{"plan":"pro","donation_tier":"beta_month_1eur","donation_reference":"anything"}`,
		`{"plan":"pro","donation_tier":"beta_year_190eur","donation_reference":"x"}`,
		`{"plan":"pro"}`,
	} {
		rr := postPlan(t, h, "u-free", body)
		if rr.Code != http.StatusPaymentRequired {
			t.Fatalf("body %s: expected 402, got %d (%s)", body, rr.Code, rr.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		if out["error_code"] != string(domain.ErrPaymentRequired) {
			t.Fatalf("expected error_code %s, got %v", domain.ErrPaymentRequired, out["error_code"])
		}
	}
	if repo.user.Plan != domain.PlanFree || repo.user.ProUntil != nil {
		t.Fatalf("user must stay free, got plan=%s pro_until=%v", repo.user.Plan, repo.user.ProUntil)
	}
}

// An active Pro user asking for Pro again gets fresh tokens and no extension.
func TestUpdatePlan_ProRequestDoesNotExtendActiveEntitlement(t *testing.T) {
	until := time.Now().UTC().Add(10 * 24 * time.Hour).Truncate(time.Second)
	repo := &authPlanMockUserRepo{user: &domain.User{ID: "u-pro", Email: "pro@example.com", Plan: domain.PlanPro, ProUntil: &until}}
	h := NewAuthHandler(repo, "secret", 24)

	rr := postPlan(t, h, "u-pro", `{"plan":"pro","donation_tier":"beta_year_190eur","donation_reference":"again"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.user.ProUntil == nil || !repo.user.ProUntil.Equal(until) {
		t.Fatalf("pro_until must not change: got %v, want %v", repo.user.ProUntil, until)
	}
}

func postAdminPlan(t *testing.T, h *AdminHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/user/plan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateUserPlan(rr, req)
	return rr
}

func TestAdminUpdateUserPlan_GrantsProForDurationAndExtends(t *testing.T) {
	repo := &authPlanMockUserRepo{user: &domain.User{ID: "u-1", Email: "u1@example.com", Plan: domain.PlanFree}}
	h := NewAdminHandler(repo, nil, nil)

	rr := postAdminPlan(t, h, `{"email":"u1@example.com","plan":"pro","duration_days":30}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.user.Plan != domain.PlanPro || repo.user.ProUntil == nil {
		t.Fatalf("expected pro with expiry, got plan=%s until=%v", repo.user.Plan, repo.user.ProUntil)
	}
	first := *repo.user.ProUntil
	if d := time.Until(first); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Fatalf("expected ~30 days, got %s", d)
	}

	rr = postAdminPlan(t, h, `{"email":"u1@example.com","plan":"pro","duration_days":30}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := repo.user.ProUntil.Sub(first); got < 29*24*time.Hour || got > 31*24*time.Hour {
		t.Fatalf("second grant should extend by ~30 days, extended by %s", got)
	}
}

func TestAdminUpdateUserPlan_DowngradeClearsExpiryAndValidatesDuration(t *testing.T) {
	until := time.Now().UTC().Add(48 * time.Hour)
	repo := &authPlanMockUserRepo{user: &domain.User{ID: "u-2", Email: "u2@example.com", Plan: domain.PlanPro, ProUntil: &until}}
	h := NewAdminHandler(repo, nil, nil)

	if rr := postAdminPlan(t, h, `{"email":"u2@example.com","plan":"pro","duration_days":-1}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("negative duration: expected 400, got %d", rr.Code)
	}
	if rr := postAdminPlan(t, h, `{"email":"u2@example.com","plan":"free"}`); rr.Code != http.StatusOK {
		t.Fatalf("downgrade: expected 200, got %d", rr.Code)
	}
	if repo.user.Plan != domain.PlanFree || repo.user.ProUntil != nil {
		t.Fatalf("expected free without expiry, got plan=%s until=%v", repo.user.Plan, repo.user.ProUntil)
	}
}
