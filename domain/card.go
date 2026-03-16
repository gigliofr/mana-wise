package domain

import "time"

// PriceSnapshot represents a historical price point for a card.
type PriceSnapshot struct {
	Date     time.Time `bson:"date"      json:"date"`
	USD      float64   `bson:"usd"       json:"usd"`
	USD_Foil float64   `bson:"usd_foil"  json:"usd_foil,omitempty"`
	EUR      float64   `bson:"eur"       json:"eur"`
}

// CardFace represents one face of a double-faced card.
type CardFace struct {
	Name      string `bson:"name"       json:"name"`
	ManaCost  string `bson:"mana_cost"  json:"mana_cost"`
	TypeLine  string `bson:"type_line"  json:"type_line"`
	OracleText string `bson:"oracle_text" json:"oracle_text"`
	Colors    []string `bson:"colors"   json:"colors"`
	CMC       float64  `bson:"cmc"      json:"cmc"`
}

// Card is the central aggregate representing a Magic card.
type Card struct {
	ID              string            `bson:"_id"              json:"id"`
	ScryfallID      string            `bson:"scryfall_id"      json:"scryfall_id"`
	Name            string            `bson:"name"             json:"name"`
	ManaCost        string            `bson:"mana_cost"        json:"mana_cost"`
	CMC             float64           `bson:"cmc"              json:"cmc"`
	TypeLine        string            `bson:"type_line"        json:"type_line"`
	OracleText      string            `bson:"oracle_text"      json:"oracle_text"`
	Colors          []string          `bson:"colors"           json:"colors"`
	ColorIdentity   []string          `bson:"color_identity"   json:"color_identity"`
	Keywords        []string          `bson:"keywords"         json:"keywords"`
	Legalities      map[string]string `bson:"legalities"       json:"legalities"`
	Rarity          string            `bson:"rarity"           json:"rarity"`
	SetCode         string            `bson:"set_code"         json:"set_code"`
	CollectorNumber string            `bson:"collector_number" json:"collector_number"`
	EdhrecRank      int               `bson:"edhrec_rank"      json:"edhrec_rank"`
	ReservedList    bool              `bson:"reserved_list"    json:"reserved_list"`
	Faces           []CardFace        `bson:"faces,omitempty"  json:"faces,omitempty"`
	PriceHistory    []PriceSnapshot   `bson:"price_history"    json:"price_history"`
	CurrentPrices   map[string]float64 `bson:"current_prices"  json:"current_prices"`
	// EmbeddingVector is a 1536-dim vector from text-embedding-3-small.
	EmbeddingVector []float64 `bson:"embedding_vector,omitempty" json:"-"`
	UpdatedAt       time.Time `bson:"updated_at"       json:"updated_at"`
}

// IsLegal returns true if the card is legal in the given format.
func (c *Card) IsLegal(format string) bool {
	status, ok := c.Legalities[format]
	return ok && status == "legal"
}

// LatestPrice returns the most recent price snapshot.
func (c *Card) LatestPrice() *PriceSnapshot {
	if len(c.PriceHistory) == 0 {
		return nil
	}
	latest := c.PriceHistory[0]
	for _, p := range c.PriceHistory[1:] {
		if p.Date.After(latest.Date) {
			latest = p
		}
	}
	return &latest
}
