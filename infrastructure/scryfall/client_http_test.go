package scryfall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, 5*time.Second, 50), srv
}

const solRingJSON = `{"id":"sol-1","name":"Sol Ring","mana_cost":"{1}","cmc":1,"type_line":"Artifact",
 "oracle_text":"{T}: Add {C}{C}.","colors":[],"color_identity":[],"legalities":{"commander":"legal"},
 "rarity":"uncommon","set":"cmm","collector_number":"400","edhrec_rank":1,"game_changer":false,
 "prices":{"usd":"1.50","usd_foil":null,"eur":"1.20"}}`

func TestGetCardByName_SendsExactQueryAndHeaders(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/named" || r.URL.Query().Get("exact") != "Sol Ring" {
			t.Errorf("unexpected request %s", r.URL.String())
		}
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("missing Scryfall-required headers: %v", r.Header)
		}
		fmt.Fprint(w, solRingJSON)
	})
	card, err := c.GetCardByName(context.Background(), "Sol Ring")
	if err != nil {
		t.Fatalf("GetCardByName: %v", err)
	}
	if card.Name != "Sol Ring" || card.CMC != 1 {
		t.Fatalf("unexpected card %+v", card)
	}
}

func TestGetCardByFuzzyNameAndSetCollector_BuildEndpoints(t *testing.T) {
	var paths []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		fmt.Fprint(w, solRingJSON)
	})
	if _, err := c.GetCardByFuzzyName(context.Background(), "sol rin"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetCardBySetCollector(context.Background(), " CMM ", "400"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetCardByID(context.Background(), "sol-1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"/cards/named?fuzzy=sol+rin", "/cards/cmm/400", "/cards/sol-1"}
	for i, w := range want {
		if paths[i] != w {
			t.Fatalf("request %d: got %s, want %s", i, paths[i], w)
		}
	}
}

func TestFetch_NotFoundIsNotRetried(t *testing.T) {
	var calls int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, `{"object":"error"}`, http.StatusNotFound)
	})
	_, err := c.GetCardByName(context.Background(), "Nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("404 must not be retried, got %d calls", calls)
	}
}

func TestFetch_RetriesServerErrorsThenSucceeds(t *testing.T) {
	var calls int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, solRingJSON)
	})
	card, err := c.GetCardByName(context.Background(), "Sol Ring")
	if err != nil || card == nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestFetch_UnexpectedStatusFailsFast(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	if _, err := c.GetCardByName(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("expected status 400 error, got %v", err)
	}
}

func TestFetch_ContextCancelledDuringBackoff(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := c.GetCardByName(ctx, "x"); err == nil {
		t.Fatal("expected an error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("backoff must stop when the context is cancelled")
	}
}

func TestSearchAllCardNames_FollowsPagination(t *testing.T) {
	var srvURL string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "is:gamechanger" {
			t.Errorf("unexpected query %q", r.URL.Query().Get("q"))
		}
		switch r.URL.Query().Get("page") {
		case "":
			json.NewEncoder(w).Encode(SearchResponse{HasMore: true, NextPage: srvURL + "/cards/search?q=is%3Agamechanger&page=2",
				Data: []ScryfallCard{{Name: "Ad Nauseam"}, {Name: "Ancient Tomb"}}})
		case "2":
			json.NewEncoder(w).Encode(SearchResponse{HasMore: false,
				Data: []ScryfallCard{{Name: "Ancient Tomb"}, {Name: "Worldly Tutor"}}})
		}
	})
	srvURL = srv.URL
	names, err := c.GameChangerNames(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "Ad Nauseam,Ancient Tomb,Worldly Tutor" {
		t.Fatalf("unexpected names %v", names)
	}
}

func TestFetchCardsByCollection_BatchesBy75(t *testing.T) {
	var batches []int
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cards/collection" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Identifiers []CollectionIdentifier `json:"identifiers"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		batches = append(batches, len(body.Identifiers))
		out := CollectionResponse{}
		for _, id := range body.Identifiers {
			out.Data = append(out.Data, ScryfallCard{Name: id.Name})
		}
		json.NewEncoder(w).Encode(out)
	})
	ids := make([]CollectionIdentifier, 160)
	for i := range ids {
		ids[i] = CollectionIdentifier{Name: fmt.Sprintf("Card %d", i)}
	}
	cards, err := c.FetchCardsByCollection(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 160 || fmt.Sprint(batches) != "[75 75 10]" {
		t.Fatalf("got %d cards in batches %v, want 160 in [75 75 10]", len(cards), batches)
	}
	empty, err := c.FetchCardsByCollection(context.Background(), nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty input: got %v, %v", empty, err)
	}
}

func TestToDomainCard_MapsFieldsPricesAndGameChanger(t *testing.T) {
	var sc ScryfallCard
	if err := json.Unmarshal([]byte(solRingJSON), &sc); err != nil {
		t.Fatal(err)
	}
	sc.GameChanger = true
	card := ToDomainCard(&sc)
	if card.ID != "sol-1" || card.ScryfallID != "sol-1" || card.SetCode != "cmm" || card.CollectorNumber != "400" {
		t.Fatalf("identity fields not mapped: %+v", card)
	}
	if !card.GameChanger {
		t.Fatal("game_changer flag must be mapped")
	}
	if card.CurrentPrices["usd"] != 1.5 || card.CurrentPrices["eur"] != 1.2 {
		t.Fatalf("prices not mapped: %v", card.CurrentPrices)
	}
	if _, ok := card.CurrentPrices["usd_foil"]; ok {
		t.Fatal("null price must not be set")
	}
	if len(card.PriceHistory) != 1 || card.PriceHistory[0].USD != 1.5 {
		t.Fatalf("price snapshot not recorded: %+v", card.PriceHistory)
	}
}
