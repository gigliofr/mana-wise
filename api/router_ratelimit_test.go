package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
)

// routerTestUserRepo is an in-memory UserRepository that counts quota increments.
type routerTestUserRepo struct {
	mu         sync.Mutex
	users      map[string]*domain.User
	increments map[string]int
}

func newRouterTestUserRepo(users ...*domain.User) *routerTestUserRepo {
	r := &routerTestUserRepo{users: map[string]*domain.User{}, increments: map[string]int{}}
	for _, u := range users {
		r.users[u.ID] = u
	}
	return r
}

func (r *routerTestUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (r *routerTestUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *routerTestUserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
	return nil
}

func (r *routerTestUserRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *routerTestUserRepo) CheckAndIncrementDailyAnalyses(_ context.Context, userID, _ string, limit int) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.increments[userID] >= limit {
		return false, nil
	}
	r.increments[userID]++
	return true, nil
}

func (r *routerTestUserRepo) incrementsFor(id string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.increments[id]
}

const routerTestSecret = "router-test-secret"

func doAnalyze(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	// An empty decklist is rejected by the handler with 400, which proves the
	// request went through rate limit and freemium gate.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", strings.NewReader(`{"decklist":"","format":"modern"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:4321"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func newRateLimitTestRouter(t *testing.T, repo domain.UserRepository) http.Handler {
	t.Helper()
	limiter := middleware.NewRateLimiter(time.Minute)
	t.Cleanup(limiter.Stop)
	return NewRouter(RouterDeps{
		UserRepo:    repo,
		JWTSecret:   routerTestSecret,
		RateLimiter: limiter,
	})
}

// Regression test for the double "/60" bug: with the router wired as in
// production, the first requests within the per-minute limit must pass.
func TestRouter_Analyze_RateLimitAllowsWithinLimitThenRejects(t *testing.T) {
	t.Setenv("RATE_LIMIT_PRO_PER_MIN", "3")
	until := time.Now().Add(24 * time.Hour)
	pro := &domain.User{ID: "u-pro", Email: "pro@example.com", Plan: domain.PlanPro, ProUntil: &until}
	repo := newRouterTestUserRepo(pro)
	h := newRateLimitTestRouter(t, repo)

	token, err := middleware.GenerateToken(pro.ID, pro.Email, string(pro.Plan), routerTestSecret, 60)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	for i := 1; i <= 3; i++ {
		rr := doAnalyze(t, h, token)
		if rr.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d: unexpected 429 within limit, body=%s", i, rr.Body.String())
		}
		if got := rr.Header().Get("X-RateLimit-Limit"); got != "3" {
			t.Fatalf("request %d: X-RateLimit-Limit=%q, want 3", i, got)
		}
	}

	rr := doAnalyze(t, h, token)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status=%d, want 429", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("4th request: missing Retry-After header")
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error_code"] != string(domain.ErrRateLimited) {
		t.Fatalf("4th request: error_code=%v, want %s", body["error_code"], domain.ErrRateLimited)
	}
}

// A request rejected by the rate limiter must not consume the free daily quota.
func TestRouter_Analyze_RateLimitedRequestDoesNotConsumeFreeQuota(t *testing.T) {
	t.Setenv("RATE_LIMIT_FREE_PER_MIN", "1")
	free := &domain.User{ID: "u-free", Email: "free@example.com", Plan: domain.PlanFree}
	repo := newRouterTestUserRepo(free)
	h := newRateLimitTestRouter(t, repo)

	token, err := middleware.GenerateToken(free.ID, free.Email, string(free.Plan), routerTestSecret, 60)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	if rr := doAnalyze(t, h, token); rr.Code == http.StatusTooManyRequests {
		t.Fatalf("first request: unexpected 429, body=%s", rr.Body.String())
	}
	if got := repo.incrementsFor(free.ID); got != 1 {
		t.Fatalf("after first request quota increments=%d, want 1", got)
	}

	rr := doAnalyze(t, h, token)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: status=%d, want 429", rr.Code)
	}
	if got := repo.incrementsFor(free.ID); got != 1 {
		t.Fatalf("rate-limited request consumed quota: increments=%d, want 1", got)
	}
}
