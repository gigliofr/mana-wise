package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

func TestCardsMetadataBatch_ReturnsItemsAndMissing(t *testing.T) {
	repo := &analyzeMockCardRepo{cardsByName: map[string]*domain.Card{
		"Lightning Bolt": {ID: "c1", Name: "Lightning Bolt", Rarity: "common", SetCode: "m11", CollectorNumber: "146"},
		"Solitude":       {ID: "c2", Name: "Solitude", Rarity: "mythic", SetCode: "mh2", CollectorNumber: "32"},
	}}
	h := NewCardsHandler(repo, nil)

	body := bytes.NewBufferString(`{"names":["Lightning Bolt","Unknown Card","Solitude"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cards/metadata/batch", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.MetadataBatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp cardMetadataBatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if len(resp.Missing) != 1 || resp.Missing[0] != "Unknown Card" {
		t.Fatalf("expected one missing card, got %+v", resp.Missing)
	}
}

func TestCardsMetadataBatch_RejectsEmpty(t *testing.T) {
	repo := &analyzeMockCardRepo{cardsByName: map[string]*domain.Card{}}
	h := NewCardsHandler(repo, nil)

	body := bytes.NewBufferString(`{"names":["   ",""]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cards/metadata/batch", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.MetadataBatch(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
