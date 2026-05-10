package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCORSMiddleware_OptionsForbidden_WhenOriginNotAllowedInProduction(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("MANAWISE_ALLOWED_ORIGINS", "")

	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCORSMiddleware_OptionsNoContent_WhenOriginAllowlisted(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("MANAWISE_ALLOWED_ORIGINS", "https://app.manawise.com,https://admin.manawise.com")

	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "https://app.manawise.com")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.manawise.com" {
		t.Fatalf("expected allow origin header to match request origin, got %q", got)
	}
}

func TestCORSMiddleware_NilNext_ReturnsServiceUnavailable(t *testing.T) {
	h := corsMiddleware(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCORSMiddleware_TypedNilNext_ReturnsInternalServerError(t *testing.T) {
	var nilMux *chi.Mux
	h := corsMiddleware(nilMux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSPAFallbackHandler_RecoversFromMalformedRequest(t *testing.T) {
	h := spaFallbackHandler(t.TempDir())
	req := &http.Request{Method: http.MethodGet, Header: make(http.Header)}
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPanicShieldMiddleware_RecoversDownstreamPanic(t *testing.T) {
	h := panicShieldMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPanicShieldMiddleware_TypedNilNext_ReturnsInternalServerError(t *testing.T) {
	var nilMux *chi.Mux
	h := panicShieldMiddleware(nilMux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSafeWriteInternalError_NilWriter(t *testing.T) {
	// Must not panic when writer is nil.
	safeWriteInternalError(nil)
}

func TestNewRouter_SharePathFallsBackToSPA(t *testing.T) {
	tmpDir := t.TempDir()
	webDistDir := filepath.Join(tmpDir, "web", "dist")
	if err := os.MkdirAll(webDistDir, 0o755); err != nil {
		t.Fatalf("create temp dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDistDir, "index.html"), []byte("<html><body>share page</body></html>"), 0o644); err != nil {
		t.Fatalf("write temp index.html: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	router := NewRouter(RouterDeps{})
	req := httptest.NewRequest(http.MethodGet, "/share/abc123", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from SPA fallback, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); got != "<html><body>share page</body></html>" {
		t.Fatalf("expected SPA index.html body, got %q", got)
	}
}
