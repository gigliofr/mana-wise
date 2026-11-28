package handlers

import (
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

func refreshCookieFrom(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == "refresh_token" {
			return c
		}
	}
	return nil
}

func cookieTestHandler(t *testing.T) (*AuthHandler, *domain.User) {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.MinCost)
	u := &domain.User{ID: "u-c", Email: "cookie@example.com", PasswordHash: string(hash), Name: "C", Plan: domain.PlanFree}
	repo := &authResetMockUserRepo{byID: map[string]*domain.User{"u-c": u}, byEmail: map[string]*domain.User{"cookie@example.com": u}}
	return NewAuthHandler(repo, "secret", 15, 60), u
}

func TestLogin_SetsHttpOnlyRefreshCookie(t *testing.T) {
	h, _ := cookieTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"cookie@example.com","password":"Password123"}`))
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "refresh_token") {
		t.Fatalf("the refresh token must not appear in the JSON body: %s", rr.Body.String())
	}
	c := refreshCookieFrom(rr)
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v1/auth" || c.MaxAge != 3600 {
		t.Fatalf("bad refresh cookie: %+v", c)
	}
	claims, err := middleware.ParseTokenClaims(c.Value, "secret")
	if err != nil || claims.TokenType != "refresh" || claims.UserID != "u-c" {
		t.Fatalf("cookie does not hold a valid refresh token: %v %+v", err, claims)
	}
}

func TestRefresh_FromCookie_RotatesCookie(t *testing.T) {
	h, _ := cookieTestHandler(t)
	old, _ := middleware.GenerateRefreshToken("u-c", "cookie@example.com", string(domain.PlanFree), "secret", 60)
	time.Sleep(1100 * time.Millisecond) // iat diverso: il nuovo token deve differire
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: old})
	rr := httptest.NewRecorder()
	h.Refresh(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rr.Code, rr.Body.String())
	}
	var payload TokenResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &payload)
	if payload.Token == "" || payload.User == nil {
		t.Fatalf("missing access token or user: %s", rr.Body.String())
	}
	c := refreshCookieFrom(rr)
	if c == nil || c.Value == "" || c.Value == old || c.Secure {
		t.Fatalf("refresh cookie not rotated (or Secure over plain HTTP): %+v", c)
	}
}

func TestRefresh_CookieWinsAndInvalidCookieIsRejected(t *testing.T) {
	h, _ := cookieTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "not-a-jwt"})
	rr := httptest.NewRecorder()
	h.Refresh(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	access, _ := middleware.GenerateToken("u-c", "cookie@example.com", string(domain.PlanFree), "secret", 15)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: access})
	rr = httptest.NewRecorder()
	h.Refresh(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("an access token must not work as refresh token, got %d", rr.Code)
	}
}

func TestLogout_ClearsCookie(t *testing.T) {
	h, _ := cookieTestHandler(t)
	rr := httptest.NewRecorder()
	h.Logout(rr, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	c := refreshCookieFrom(rr)
	if rr.Code != http.StatusOK || c == nil || c.MaxAge != -1 || c.Path != "/api/v1/auth" {
		t.Fatalf("logout must clear the cookie: %d %+v", rr.Code, c)
	}
}
