package llm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/infrastructure/cache"
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
	return c.provider
}

// Model returns the model name configured for this connector.
func (c *Connector) Model() string {
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
func (c *Connector) Suggestions(ctx context.Context, decklistHash, format string, analysis *domain.AnalysisResult) (string, error) {
	cacheKey := fmt.Sprintf("%s::%s", decklistHash, format)
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(string), nil
	}

	prompt := buildPrompt(format, analysis)

	llmCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt(),
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

func systemPrompt() string {
	return `You are ManaWise AI, a professional Magic: The Gathering deck analyst.
You receive deterministic analysis data for a deck and provide concise, actionable improvement suggestions.
Rules:
- Focus on the 3 most impactful improvements.
- Reference specific card categories or slots.
- Keep each suggestion to 1-2 sentences.
- Use plain text (no markdown headers).
- Do not repeat information already provided in the analysis.`
}

func buildPrompt(format string, a *domain.AnalysisResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Format: %s\n", format))
	sb.WriteString(fmt.Sprintf("Total cards: %d, Average CMC: %.2f, Land count: %d (ideal: %d)\n",
		a.Mana.TotalCards, a.Mana.AverageCMC, a.Mana.LandCount, a.Mana.IdealLandCount))

	sb.WriteString("Mana curve distribution:\n")
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

	sb.WriteString("\nProvide 3 targeted improvement suggestions for this deck:")
	return sb.String()
}
