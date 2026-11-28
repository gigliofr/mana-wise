package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/api/middleware"
)

// The web client reads the new deck id only from a JSON response: the
// Content-Type must survive the 201 status, and the maybeboard must persist.
func TestDeckCreate_ReturnsJSONWithIDAndMaybeboard(t *testing.T) {
	repo := &legalityMockDeckRepo{}
	h := NewDeckHandler(repo, nil, nil, nil, nil, nil)

	body := `{"name":"Silver","format":"commander","cards":[{"card_name":"The First Sliver","quantity":1,"is_commander":true},{"card_name":"Forest","quantity":14}],"maybeboard":[{"card_name":"Sliver Hive","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1"))
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Result().Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	var resp struct {
		ID         string `json:"id"`
		Maybeboard []struct {
			CardName string `json:"card_name"`
		} `json:"maybeboard"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Fatal("expected deck id in response")
	}
	if len(resp.Maybeboard) != 1 || resp.Maybeboard[0].CardName != "Sliver Hive" {
		t.Fatalf("maybeboard not returned: %+v", resp.Maybeboard)
	}
	if repo.deck == nil || len(repo.deck.Maybeboard) != 1 {
		t.Fatal("maybeboard not persisted")
	}
}
