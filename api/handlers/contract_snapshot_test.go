package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/cache"
	"github.com/gigliofr/mana-wise/usecase"
	"github.com/go-chi/chi/v5"
)

// Contract snapshot tests: they pin the *shape* of the JSON responses (field
// paths and value types, not the values), so that a change that renames or
// drops a field the frontend relies on fails loudly.
//
// Regenerate the golden files after an intentional change with:
//
//	go test ./api/handlers -run Contract -update
var updateGolden = flag.Bool("update", false, "rewrite contract golden files")

// jsonShape flattens a decoded JSON value into sorted "path: type" lines.
// Arrays contribute the union of their elements' shapes under "path[]".
func jsonShape(v any) []string {
	set := map[string]bool{}
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			set[path+": object"] = true
			for k, child := range x {
				walk(path+"."+k, child)
			}
		case []any:
			set[path+": array"] = true
			for _, child := range x {
				walk(path+"[]", child)
			}
		case string:
			set[path+": string"] = true
		case float64:
			set[path+": number"] = true
		case bool:
			set[path+": bool"] = true
		case nil:
			set[path+": null"] = true
		default:
			set[fmt.Sprintf("%s: %T", path, v)] = true
		}
	}
	walk("$", v)
	out := make([]string, 0, len(set))
	for line := range set {
		out = append(out, line)
	}
	sort.Strings(out)
	return out
}

func assertContract(t *testing.T, name string, body []byte) {
	t.Helper()
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("%s: invalid JSON: %v\n%s", name, err, body)
	}
	got := strings.Join(jsonShape(decoded), "\n") + "\n"
	path := filepath.Join("testdata", "contract", name+".golden")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: missing golden file %s (run with -update): %v", name, path, err)
	}
	if string(want) != got {
		t.Fatalf("%s: response shape changed.\n%s\nIf intentional, run: go test ./api/handlers -run Contract -update",
			name, shapeDiff(strings.Split(strings.TrimSpace(string(want)), "\n"), strings.Split(strings.TrimSpace(got), "\n")))
	}
}

func shapeDiff(want, got []string) string {
	w, g := map[string]bool{}, map[string]bool{}
	for _, l := range want {
		w[l] = true
	}
	for _, l := range got {
		g[l] = true
	}
	var b strings.Builder
	for _, l := range want {
		if !g[l] {
			b.WriteString("- " + l + "\n")
		}
	}
	for _, l := range got {
		if !w[l] {
			b.WriteString("+ " + l + "\n")
		}
	}
	return b.String()
}

func contractCommanderCards() map[string]*domain.Card {
	return map[string]*domain.Card{
		"Sol Ring":      {ID: "c-sol", Name: "Sol Ring", CMC: 1, TypeLine: "Artifact", OracleText: "{T}: Add {C}{C}.", Colors: []string{}, Legalities: map[string]string{"commander": "legal"}},
		"Rhystic Study": {ID: "c-rhystic", Name: "Rhystic Study", CMC: 3, TypeLine: "Enchantment", OracleText: "Whenever an opponent casts a spell, you may draw a card unless that player pays {1}.", Colors: []string{"U"}, Legalities: map[string]string{"commander": "legal"}},
		"Island":        {ID: "c-island", Name: "Island", CMC: 0, TypeLine: "Basic Land - Island", Colors: []string{}, Legalities: map[string]string{"commander": "legal"}},
	}
}

func TestContract_AnalyzeCommander(t *testing.T) {
	repo := &analyzeMockCardRepo{cardsByName: contractCommanderCards()}
	uc := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, repo, 2)
	bracketUC := usecase.NewCommanderBracketUseCase(nil)
	h := NewAnalyzeHandler(uc, nil, bracketUC, &analyzeMockUserRepo{}, &mockTracker{})

	body, _ := json.Marshal(map[string]string{
		"format":   "commander",
		"decklist": "1 Sol Ring\n1 Rhystic Study\n98 Island",
		"locale":   "en",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-contract")
	ctx = context.WithValue(ctx, middleware.ContextKeyPlan, string(domain.PlanFree))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	assertContract(t, "analyze_commander", rr.Body.Bytes())

	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	bracket, _ := resp["commander_bracket"].(map[string]any)
	if bracket == nil || bracket["label"] != "Upgraded" {
		t.Fatalf("expected commander_bracket Upgraded (1 Game Changer), got %v", resp["commander_bracket"])
	}
}

func TestContract_Score(t *testing.T) {
	cards := []*domain.Card{
		{ID: "land", Name: "Test Land", CMC: 0, TypeLine: "Basic Land", EdhrecRank: 100,
			PriceHistory: []domain.PriceSnapshot{{Date: time.Unix(0, 0), USD: 0.5}}},
		{ID: "spell", Name: "Test Spell", CMC: 3, TypeLine: "Creature — Test", OracleText: "Test ability", EdhrecRank: 500,
			PriceHistory: []domain.PriceSnapshot{{Date: time.Unix(0, 0), USD: 5}}},
	}
	reqBytes, _ := json.Marshal(map[string]any{"decklist": cards, "format": "commander"})
	req := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	impactUC := &usecase.ImpactScoreUseCase{Weights: domain.DefaultImpactWeights()}
	handler := newScoreHandler(usecase.NewScoreUseCase(impactUC, &usecase.PowerLevelUseCase{}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	assertContract(t, "score", rr.Body.Bytes())
}

func TestContract_SharePDF(t *testing.T) {
	repo := newMemorySharedAnalysisRepo()
	deckRepo := &pdfMockDeckRepo{deck: &domain.Deck{
		ID:     "deck-contract",
		Format: "commander",
		Cards:  []domain.DeckCard{{CardName: "Sol Ring", Quantity: 1}, {CardName: "Rhystic Study", Quantity: 1}, {CardName: "Island", Quantity: 98}},
	}}
	cardRepo := &analyzeMockCardRepo{cardsByName: contractCommanderCards()}
	analyzeUC := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, cardRepo, 2)
	if _, err := usecase.ShareAnalysis(context.Background(), repo,
		usecase.ShareAnalysisRequest{DeckID: "deck-contract", Channel: "link", TTL: usecase.DefaultShareLinkTTL},
		"https://example.test"); err != nil {
		t.Fatalf("create share link: %v", err)
	}
	var token string
	for id := range repo.links {
		token = id
	}

	h := NewPublicSharePDFHandler(repo, deckRepo, analyzeUC)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analysis/share/"+token+"/pdf", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type=%q, want application/pdf", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") || !strings.Contains(cd, ".pdf") {
		t.Fatalf("Content-Disposition=%q, want inline with a .pdf filename", cd)
	}
	pdf, _ := io.ReadAll(rr.Body)
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatal("response is not a complete PDF document")
	}
	pages := bytes.Count(pdf, []byte("/Type /Page\n")) + bytes.Count(pdf, []byte("/Type /Page>>")) + bytes.Count(pdf, []byte("/Type /Page "))
	if pages < 1 {
		t.Fatalf("expected at least one page, found %d", pages)
	}
}

func TestSharePDF_CachedPerDeckVersion(t *testing.T) {
	renders := 0
	orig := renderSharePDF
	renderSharePDF = func(b *sharedAnalysisBundle, url string) ([]byte, error) {
		renders++
		return orig(b, url)
	}
	t.Cleanup(func() { renderSharePDF = orig })

	repo := newMemorySharedAnalysisRepo()
	deck := &domain.Deck{ID: "deck-cache", Format: "commander", Version: 1, UpdatedAt: time.Unix(1000, 0),
		Cards: []domain.DeckCard{{CardName: "Sol Ring", Quantity: 1}, {CardName: "Island", Quantity: 99}}}
	deckRepo := &pdfMockDeckRepo{deck: deck}
	cardRepo := &analyzeMockCardRepo{cardsByName: contractCommanderCards()}
	analyzeUC := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, cardRepo, 2)
	if _, err := usecase.ShareAnalysis(context.Background(), repo,
		usecase.ShareAnalysisRequest{DeckID: deck.ID, Channel: "link", TTL: usecase.DefaultShareLinkTTL}, "https://example.test"); err != nil {
		t.Fatal(err)
	}
	var token string
	for id := range repo.links {
		token = id
	}
	h := NewPublicSharePDFHandler(repo, deckRepo, analyzeUC).WithPDFCache(cache.New())
	get := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analysis/share/"+token+"/pdf", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("token", token)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !bytes.HasPrefix(rr.Body.Bytes(), []byte("%PDF")) {
			t.Fatalf("status %d: not a PDF", rr.Code)
		}
		return rr.Code
	}

	get()
	get()
	if renders != 1 {
		t.Fatalf("second open of the same version must be served from cache, renders=%d", renders)
	}
	deck.Version = 2
	deck.UpdatedAt = time.Unix(2000, 0)
	get()
	if renders != 2 {
		t.Fatalf("a new deck version must be rendered again, renders=%d", renders)
	}
}
