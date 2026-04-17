package scryfall

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

const (
	maxRetries           = 3
	retryBaseMs          = 200 // base delay in milliseconds for exponential backoff
	minTooManyReqBackoff = 1 * time.Second
	maxRetryBackoff      = 10 * time.Second
)

// ScryfallCard is the raw shape returned by the Scryfall API.
type ScryfallCard struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	ManaCost        string            `json:"mana_cost"`
	CMC             float64           `json:"cmc"`
	TypeLine        string            `json:"type_line"`
	OracleText      string            `json:"oracle_text"`
	Colors          []string          `json:"colors"`
	ColorIdentity   []string          `json:"color_identity"`
	Keywords        []string          `json:"keywords"`
	Legalities      map[string]string `json:"legalities"`
	Rarity          string            `json:"rarity"`
	Set             string            `json:"set"`
	CollectorNumber string            `json:"collector_number"`
	EdhrecRank      int               `json:"edhrec_rank"`
	ReservedList    bool              `json:"reserved_list"`
	Layout          string            `json:"layout"`
	Prices          struct {
		USD     *string `json:"usd"`
		USDFoil *string `json:"usd_foil"`
		EUR     *string `json:"eur"`
	} `json:"prices"`
	CardFaces []struct {
		Name       string   `json:"name"`
		ManaCost   string   `json:"mana_cost"`
		TypeLine   string   `json:"type_line"`
		OracleText string   `json:"oracle_text"`
		Colors     []string `json:"colors"`
		CMC        float64  `json:"cmc"`
	} `json:"card_faces"`
}

// SearchResponse is the paginated response from /cards/search.
type SearchResponse struct {
	TotalCards int            `json:"total_cards"`
	HasMore    bool           `json:"has_more"`
	NextPage   string         `json:"next_page"`
	Data       []ScryfallCard `json:"data"`
}

// Client is a rate-limited, retrying Scryfall API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	limiter    *rateLimiter
}

// rateLimiter allows at most N requests per second.
type rateLimiter struct {
	mu       sync.Mutex
	tokens   int
	maxRate  int
	lastTick time.Time
}

func newRateLimiter(rps int) *rateLimiter {
	return &rateLimiter{
		tokens:   rps,
		maxRate:  rps,
		lastTick: time.Now(),
	}
}

func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(r.lastTick)
		if elapsed >= time.Second {
			r.tokens = r.maxRate
			r.lastTick = now
		}
		if r.tokens > 0 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}
		r.mu.Unlock()

		// Wait until next token is available.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// NewClient creates a Scryfall client.
func NewClient(baseURL string, timeout time.Duration, rateLimit int) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		limiter:    newRateLimiter(rateLimit),
	}
}

// GetCardByID fetches a card by its Scryfall UUID.
func (c *Client) GetCardByID(ctx context.Context, id string) (*ScryfallCard, error) {
	endpoint := fmt.Sprintf("%s/cards/%s", c.baseURL, url.PathEscape(id))
	return c.fetchCard(ctx, endpoint)
}

// GetCardByName fetches a card by exact name using /cards/named.
func (c *Client) GetCardByName(ctx context.Context, name string) (*ScryfallCard, error) {
	endpoint := fmt.Sprintf("%s/cards/named?exact=%s", c.baseURL, url.QueryEscape(name))
	return c.fetchCard(ctx, endpoint)
}

// GetCardByFuzzyName fetches the best fuzzy match using /cards/named?fuzzy=.
func (c *Client) GetCardByFuzzyName(ctx context.Context, name string) (*ScryfallCard, error) {
	endpoint := fmt.Sprintf("%s/cards/named?fuzzy=%s", c.baseURL, url.QueryEscape(name))
	return c.fetchCard(ctx, endpoint)
}

// GetCardBySetCollector fetches a card by set code + collector number using /cards/{set}/{collector}.
func (c *Client) GetCardBySetCollector(ctx context.Context, setCode, collectorNumber string) (*ScryfallCard, error) {
	endpoint := fmt.Sprintf("%s/cards/%s/%s", c.baseURL, url.PathEscape(strings.ToLower(strings.TrimSpace(setCode))), url.PathEscape(strings.ToLower(strings.TrimSpace(collectorNumber))))
	return c.fetchCard(ctx, endpoint)
}

// SearchCards queries /cards/search and returns a paginated result.
func (c *Client) SearchCards(ctx context.Context, query string) (*SearchResponse, error) {
	endpoint := fmt.Sprintf("%s/cards/search?q=%s", c.baseURL, url.QueryEscape(query))

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	body, err := c.doWithRetry(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("SearchCards unmarshal: %w", err)
	}
	return &result, nil
}

// CollectionIdentifier represents a card to fetch by ID or name.
type CollectionIdentifier struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Set  string `json:"set,omitempty"`
}

// CollectionResponse is returned from POST /cards/collection.
type CollectionResponse struct {
	Data []ScryfallCard `json:"data"`
}

// FetchCardsByCollection fetches multiple cards by ID or name using POST /cards/collection.
// The API accepts up to 75 identifiers per request.
func (c *Client) FetchCardsByCollection(ctx context.Context, identifiers []CollectionIdentifier) ([]ScryfallCard, error) {
	if len(identifiers) == 0 {
		return []ScryfallCard{}, nil
	}

	// Scryfall collection API accepts max 75 identifiers per request
	const maxBatch = 75
	var allCards []ScryfallCard

	for i := 0; i < len(identifiers); i += maxBatch {
		end := i + maxBatch
		if end > len(identifiers) {
			end = len(identifiers)
		}

		batch := identifiers[i:end]
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		endpoint := fmt.Sprintf("%s/cards/collection", c.baseURL)
		payload := map[string]interface{}{
			"identifiers": batch,
		}

		body, err := c.doPostWithRetry(ctx, endpoint, payload)
		if err != nil {
			return nil, fmt.Errorf("FetchCardsByCollection batch %d-%d: %w", i, end, err)
		}

		var result CollectionResponse
		if err = json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("FetchCardsByCollection unmarshal batch %d-%d: %w", i, end, err)
		}

		allCards = append(allCards, result.Data...)
	}

	return allCards, nil
}

// fetchCard performs a GET and deserialises a single ScryfallCard.
func (c *Client) fetchCard(ctx context.Context, endpoint string) (*ScryfallCard, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	body, err := c.doWithRetry(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var card ScryfallCard
	if err = json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("fetchCard unmarshal: %w", err)
	}
	return &card, nil
}

func parseRetryAfter(header string) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	if secs, err := time.ParseDuration(header + "s"); err == nil {
		if secs < 0 {
			return 0, false
		}
		return secs, true
	}

	if at, err := http.ParseTime(header); err == nil {
		d := time.Until(at)
		if d < 0 {
			return 0, true
		}
		return d, true
	}

	return 0, false
}

func computeRetryDelay(statusCode int, attempt int, retryAfterHeader string) time.Duration {
	base := time.Duration(float64(retryBaseMs)*math.Pow(2, float64(attempt))) * time.Millisecond
	if statusCode == http.StatusTooManyRequests && base < minTooManyReqBackoff {
		base = minTooManyReqBackoff
	}

	if retryAfter, ok := parseRetryAfter(retryAfterHeader); ok && retryAfter > base {
		base = retryAfter
	}

	if base > maxRetryBackoff {
		return maxRetryBackoff
	}
	return base
}

func waitRetryDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// doWithRetry executes a GET with exponential backoff on 429 / 5xx responses.
func (c *Client) doWithRetry(ctx context.Context, endpoint string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", "ManaWise/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do (attempt %d): %w", attempt+1, err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read body (attempt %d): %w", attempt+1, readErr)
			continue
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			return body, nil
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", endpoint)
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("http %d on attempt %d", resp.StatusCode, attempt+1)
			if attempt == maxRetries {
				continue
			}
			delay := computeRetryDelay(resp.StatusCode, attempt, resp.Header.Get("Retry-After"))
			if err := waitRetryDelay(ctx, delay); err != nil {
				return nil, err
			}
			continue
		default:
			return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, endpoint)
		}
	}

	return nil, fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
}

// doPostWithRetry executes a POST with JSON body and exponential backoff on 429 / 5xx responses.
func (c *Client) doPostWithRetry(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", "ManaWise/1.0")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do (attempt %d): %w", attempt+1, err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read body (attempt %d): %w", attempt+1, readErr)
			continue
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			return body, nil
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", endpoint)
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("http %d on attempt %d", resp.StatusCode, attempt+1)
			if attempt == maxRetries {
				continue
			}
			delay := computeRetryDelay(resp.StatusCode, attempt, resp.Header.Get("Retry-After"))
			if err := waitRetryDelay(ctx, delay); err != nil {
				return nil, err
			}
			continue
		default:
			return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, endpoint)
		}
	}

	return nil, fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
}

// ToDomainCard converts a ScryfallCard to a domain.Card.
func ToDomainCard(sc *ScryfallCard) *domain.Card {
	card := &domain.Card{
		ID:              sc.ID,
		ScryfallID:      sc.ID,
		Name:            sc.Name,
		ManaCost:        sc.ManaCost,
		CMC:             sc.CMC,
		TypeLine:        sc.TypeLine,
		OracleText:      sc.OracleText,
		Colors:          sc.Colors,
		ColorIdentity:   sc.ColorIdentity,
		Keywords:        sc.Keywords,
		Legalities:      sc.Legalities,
		Rarity:          sc.Rarity,
		SetCode:         sc.Set,
		CollectorNumber: sc.CollectorNumber,
		EdhrecRank:      sc.EdhrecRank,
		ReservedList:    sc.ReservedList,
		Layout:          sc.Layout,
		UpdatedAt:       time.Now().UTC(),
	}

	// Faces
	for _, f := range sc.CardFaces {
		card.Faces = append(card.Faces, domain.CardFace{
			Name:       f.Name,
			ManaCost:   f.ManaCost,
			TypeLine:   f.TypeLine,
			OracleText: f.OracleText,
			Colors:     f.Colors,
			CMC:        f.CMC,
		})
	}

	// Current prices snapshot
	card.CurrentPrices = make(map[string]float64)
	snapshot := domain.PriceSnapshot{Date: time.Now().UTC()}
	if sc.Prices.USD != nil {
		var v float64
		fmt.Sscanf(*sc.Prices.USD, "%f", &v)
		card.CurrentPrices["usd"] = v
		snapshot.USD = v
	}
	if sc.Prices.USDFoil != nil {
		var v float64
		fmt.Sscanf(*sc.Prices.USDFoil, "%f", &v)
		card.CurrentPrices["usd_foil"] = v
		snapshot.USD_Foil = v
	}
	if sc.Prices.EUR != nil {
		var v float64
		fmt.Sscanf(*sc.Prices.EUR, "%f", &v)
		card.CurrentPrices["eur"] = v
		snapshot.EUR = v
	}
	card.PriceHistory = []domain.PriceSnapshot{snapshot}

	return card
}
