package llm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/cache"
	openai "github.com/sashabaranov/go-openai"
)

// Connector wraps an LLM provider with result caching.
type Connector struct {
	provider  string
	client    *openai.Client
	model     string
	maxTokens int
	cacheTTL  time.Duration
	cache     *cache.Cache
	timeout   time.Duration
}

// Provider returns the normalized provider name configured for this connector.
func (c *Connector) Provider() string {
	if c == nil {
		return ""
	}
	return c.provider
}

// Model returns the model name configured for this connector.
func (c *Connector) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

// NewConnector creates a new LLM connector backed by OpenAI.
func NewConnector(provider, apiKey, baseURL, model string, maxTokens int, timeout, cacheTTL time.Duration) *Connector {
	clientCfg := openai.DefaultConfig(apiKey)
	if strings.TrimSpace(baseURL) != "" {
		clientCfg.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}

	c := &Connector{
		provider:  provider,
		client:    openai.NewClientWithConfig(clientCfg),
		model:     model,
		maxTokens: maxTokens,
		cacheTTL:  cacheTTL,
		cache:     cache.New(),
		timeout:   timeout,
	}
	return c
}

// Suggestions sends a structured analysis to the LLM and returns AI suggestions.
// Results are cached by (decklistHash, format) for cacheTTL duration.
func (c *Connector) Suggestions(ctx context.Context, decklistHash, decklist, format, locale string, analysis *domain.AnalysisResult) (string, error) {
	if c == nil {
		return "", fmt.Errorf("LLM Suggestions: connector unavailable")
	}

	cacheKey := fmt.Sprintf("%s::%s::%s", decklistHash, format, normalizeLocale(locale))
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(string), nil
	}

	prompt := buildPrompt(format, normalizeLocale(locale), decklist, analysis)

	llmCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt(normalizeLocale(locale)),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: 0.4,
	}

	resp, err := c.client.CreateChatCompletion(llmCtx, req)
	if err != nil {
		return "", fmt.Errorf("LLM Suggestions: %s", humanizeProviderError(c.provider, err))
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	c.cache.Set(cacheKey, text, c.cacheTTL)
	return text, nil
}

// EmbedText generates a single embedding vector using text-embedding-3-small.
func (c *Connector) EmbedText(ctx context.Context, input string) ([]float64, error) {
	if c == nil {
		return nil, fmt.Errorf("LLM EmbedText: connector unavailable")
	}

	llmCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CreateEmbeddings(llmCtx, openai.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []string{input},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM EmbedText: %s", humanizeProviderError(c.provider, err))
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("LLM EmbedText: empty response")
	}

	f32 := resp.Data[0].Embedding
	vector := make([]float64, len(f32))
	for i := range f32 {
		vector[i] = float64(f32[i])
	}
	return vector, nil
}

// HashDecklist produces a stable SHA-256 hex hash of a raw decklist string.
func HashDecklist(decklist, format string) string {
	h := sha256.Sum256([]byte(decklist + "|" + format))
	return fmt.Sprintf("%x", h)
}

func humanizeProviderError(provider string, err error) string {
	if err == nil {
		return "unknown provider error"
	}

	raw := err.Error()
	status := extractStatusCode(raw)
	lower := strings.ToLower(raw)

	if strings.Contains(raw, "cannot unmarshal array into Go value of type openai.ErrorResponse") {
		switch status {
		case 404:
			return "provider returned 404 (model not found). Verify LLM_MODEL for the selected provider."
		case 429:
			return "provider quota exceeded (429). Check Gemini/OpenAI quota and retry later."
		default:
			if status > 0 {
				return fmt.Sprintf("provider returned status %d with a non-standard error payload", status)
			}
			return "provider returned a non-standard error payload"
		}
	}

	if status == 429 || strings.Contains(lower, "quota") || strings.Contains(lower, "resource_exhausted") {
		return "provider quota exceeded (429). Check Gemini/OpenAI quota and retry later."
	}

	return raw
}

func extractStatusCode(msg string) int {
	re := regexp.MustCompile(`status code: (\d{3})`)
	m := re.FindStringSubmatch(msg)
	if len(m) != 2 {
		return 0
	}
	code, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return code
}

func systemPrompt(locale string) string {
	if locale == "it" {
		return `Sei ManaWise AI, un analista professionale di mazzi Magic: The Gathering.
Ricevi l'analisi deterministica e la lista completa del mazzo e devi fornire suggerimenti operativi.
Regole:
- Rispondi esclusivamente in italiano.
- Fornisci esattamente 3 suggerimenti numerati.
- Ogni suggerimento deve avere questo formato: "TOGLI: <carta/e esatte dal mazzo> METTI: <carta/e specifiche legali nel formato> PERCHE': <motivo conciso>".
- In TOGLI nomina carte specifiche presenti nel mazzo fornito.
- In METTI suggerisci carte reali con il loro nome esatto, legali nel formato richiesto.
- Non usare placeholder generici come 'una carta' o 'slot'; usa sempre nomi reali.
- Rimani conciso (1-2 frasi per suggerimento), senza markdown o titoli.`
	}
	return `You are ManaWise AI, a professional Magic: The Gathering deck analyst.
You receive deterministic analysis data and the full decklist, and provide concise, actionable improvement suggestions.
Rules:
- Respond only in English.
- Provide exactly 3 numbered suggestions.
- Each suggestion must use this format: "CUT: <exact card name(s) from the deck> ADD: <specific real card name(s) legal in the format> WHY: <concise reason>".
- In CUT, name specific cards present in the provided decklist.
- In ADD, suggest real Magic cards by their exact name, legal in the requested format.
- Never use generic placeholders like 'a card' or 'slot'; always use real card names.
- Keep each suggestion to 1-2 sentences and use plain text (no markdown headers).`
}

func buildPrompt(format, locale, decklist string, a *domain.AnalysisResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Format: %s\n", format))
	sb.WriteString(fmt.Sprintf("Language: %s\n", locale))
	sb.WriteString(fmt.Sprintf("Total cards: %d, Average CMC: %.2f, Land count: %d (ideal: %d)\n",
		a.Mana.TotalCards, a.Mana.AverageCMC, a.Mana.LandCount, a.Mana.IdealLandCount))

	if strings.TrimSpace(decklist) != "" {
		sb.WriteString("\nCurrent decklist (use these exact card names in your suggestions):\n")
		for _, line := range strings.Split(strings.TrimSpace(decklist), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}

	sb.WriteString("\nMana curve distribution:\n")
	for _, b := range a.Mana.Distribution {
		sb.WriteString(fmt.Sprintf("  CMC %d: %d cards\n", b.CMC, b.Count))
	}

	if len(a.Mana.Suggestions) > 0 {
		sb.WriteString("Mana curve issues found:\n")
		for _, s := range a.Mana.Suggestions {
			sb.WriteString(fmt.Sprintf("  - [%s] %s\n", s.Urgency, s.Reason))
		}
	}

	sb.WriteString(fmt.Sprintf("Interaction score: %.1f/100\n", a.Interaction.TotalScore))
	for _, bd := range a.Interaction.Breakdowns {
		sb.WriteString(fmt.Sprintf("  %s: %d (ideal %d)\n", bd.Category, bd.Count, bd.Ideal))
	}

	sb.WriteString("\nProvide 3 targeted swap suggestions naming specific cards from the decklist above. Each must state exactly which card(s) to remove and which specific card(s) to add as replacements. Ensure format legality for all additions.")
	return sb.String()
}

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "it") {
		return "it"
	}
	return "en"
}
