package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gigliofr/mana-wise/domain"
)

type analyzeRequest struct {
	Decklist string `json:"decklist"`
	Format   string `json:"format"`
}

type analyzeResponse struct {
	Deterministic struct {
		Mana struct {
			TotalCards     int     `json:"total_cards"`
			AverageCMC     float64 `json:"average_cmc"`
			LandCount      int     `json:"land_count"`
			IdealLandCount int     `json:"ideal_land_count"`
		} `json:"mana"`
		Interaction struct {
			TotalScore float64 `json:"total_score"`
		} `json:"interaction"`
	} `json:"deterministic"`
	AISuggestions string `json:"ai_suggestions"`
	LatencyMs     int64  `json:"latency_ms"`
}

type cardLookupResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type priceTrendResponse struct {
	CardID     string   `json:"card_id"`
	CardName   string   `json:"card_name"`
	Current    float64  `json:"current_usd"`
	Change7d   *float64 `json:"change_7d_pct,omitempty"`
	Change30d  *float64 `json:"change_30d_pct,omitempty"`
	Change90d  *float64 `json:"change_90d_pct,omitempty"`
	SpikeAlert bool     `json:"spike_alert"`
}

type synergyItem struct {
	Card struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"card"`
	Score float64 `json:"score"`
}

type botConfig struct {
	DiscordToken string
	APIURL       string
	APIJWT       string
	DefaultFmt   string
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord init: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handleMessage(s, m, cfg)
	})

	if err = dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	log.Println("✅ ManaWise Discord bot online")
	log.Println("Commands: !analizza, !prezzo, !sinergie")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 bot shutdown")
}

func loadConfig() (*botConfig, error) {
	cfg := &botConfig{
		DiscordToken: strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")),
		APIURL:       strings.TrimRight(strings.TrimSpace(os.Getenv("MANAWISE_API_URL")), "/"),
		APIJWT:       strings.TrimSpace(os.Getenv("MANAWISE_BOT_JWT")),
		DefaultFmt:   strings.TrimSpace(os.Getenv("BOT_DEFAULT_FORMAT")),
	}
	if cfg.DefaultFmt == "" {
		cfg.DefaultFmt = "commander"
	}
	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	if cfg.APIJWT == "" {
		return nil, fmt.Errorf("MANAWISE_BOT_JWT is required")
	}
	return cfg, nil
}

func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate, cfg *botConfig) {
	if m.Author == nil || m.Author.Bot {
		return
	}
	content := strings.TrimSpace(m.Content)
	switch {
	case strings.HasPrefix(strings.ToLower(content), "!analizza"):
		handleAnalyzeCommand(s, m.ChannelID, content, cfg)
	case strings.HasPrefix(strings.ToLower(content), "!prezzo"):
		handlePriceCommand(s, m.ChannelID, content, cfg)
	case strings.HasPrefix(strings.ToLower(content), "!sinergie"):
		handleSynergiesCommand(s, m.ChannelID, content, cfg)
	}
}

func handleAnalyzeCommand(s *discordgo.Session, channelID, content string, cfg *botConfig) {
	format, decklist := parseAnalyzeCommand(content, cfg.DefaultFmt)
	if !domain.IsValidFormat(format) {
		_, _ = s.ChannelMessageSend(channelID, fmt.Sprintf("Formato non supportato: `%s`", format))
		return
	}
	if strings.TrimSpace(decklist) == "" {
		_, _ = s.ChannelMessageSend(channelID,
			"Uso: `!analizza [format]` seguito da decklist su nuove righe.\nEsempio:\n`!analizza modern\n4 Lightning Bolt\n4 Goblin Guide`")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := callAnalyze(ctx, cfg, decklist, format)
	if err != nil {
		_, _ = s.ChannelMessageSend(channelID, fmt.Sprintf("Errore analisi: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔮 ManaWise Deck Analysis",
		Description: fmt.Sprintf("Formato: **%s**", titleCase(format)),
		Color:       0x7C5CBF,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Carte", Value: fmt.Sprintf("%d", result.Deterministic.Mana.TotalCards), Inline: true},
			{Name: "Avg CMC", Value: fmt.Sprintf("%.2f", result.Deterministic.Mana.AverageCMC), Inline: true},
			{Name: "Lands", Value: fmt.Sprintf("%d (ideal %d)", result.Deterministic.Mana.LandCount, result.Deterministic.Mana.IdealLandCount), Inline: true},
			{Name: "Interaction", Value: fmt.Sprintf("%.1f / 100", result.Deterministic.Interaction.TotalScore), Inline: true},
			{Name: "Latency", Value: fmt.Sprintf("%d ms", result.LatencyMs), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "ManaWise AI"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if strings.TrimSpace(result.AISuggestions) != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "AI Suggestions", Value: truncate(result.AISuggestions, 900)})
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func handlePriceCommand(s *discordgo.Session, channelID, content string, cfg *botConfig) {
	cardName := commandRemainder(content, "!prezzo")
	if cardName == "" {
		_, _ = s.ChannelMessageSend(channelID, "Uso: `!prezzo Nome Carta`")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	trend, err := callPriceTrendByName(ctx, cfg, cardName)
	if err != nil {
		_, _ = s.ChannelMessageSend(channelID, fmt.Sprintf("Errore prezzo: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💰 ManaWise Price Trend",
		Description: fmt.Sprintf("**%s**", trend.CardName),
		Color:       0xE5A22A,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Current USD", Value: fmt.Sprintf("$%.2f", trend.Current), Inline: true},
			{Name: "7d", Value: formatPct(trend.Change7d), Inline: true},
			{Name: "30d", Value: formatPct(trend.Change30d), Inline: true},
			{Name: "90d", Value: formatPct(trend.Change90d), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "ManaWise AI"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if trend.SpikeAlert {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Alert", Value: "Price spike detected (>20% in 7d)"})
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func handleSynergiesCommand(s *discordgo.Session, channelID, content string, cfg *botConfig) {
	cardName := commandRemainder(content, "!sinergie")
	if cardName == "" {
		_, _ = s.ChannelMessageSend(channelID, "Uso: `!sinergie Nome Carta`")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	synergies, resolvedName, err := callSynergiesByName(ctx, cfg, cardName, 5)
	if err != nil {
		_, _ = s.ChannelMessageSend(channelID, fmt.Sprintf("Errore sinergie: %v", err))
		return
	}
	if len(synergies) == 0 {
		_, _ = s.ChannelMessageSend(channelID, "Nessuna sinergia trovata. Verifica che gli embedding siano stati generati con `/api/v1/embed/batch`.")
		return
	}

	lines := make([]string, 0, len(synergies))
	for i, item := range synergies {
		lines = append(lines, fmt.Sprintf("%d. %s (%.3f)", i+1, item.Card.Name, item.Score))
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🧩 ManaWise Synergies",
		Description: fmt.Sprintf("Top 5 sinergie per **%s**", resolvedName),
		Color:       0x3ECF6E,
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "Cards",
			Value: strings.Join(lines, "\n"),
		}},
		Footer:    &discordgo.MessageEmbedFooter{Text: "ManaWise AI"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func parseAnalyzeCommand(content, defaultFormat string) (string, string) {
	lines := strings.Split(content, "\n")
	header := strings.Fields(strings.TrimSpace(lines[0]))
	format := defaultFormat
	if len(header) >= 2 {
		format = domain.NormalizeFormat(header[1])
	}
	decklist := ""
	if len(lines) > 1 {
		decklist = strings.Join(lines[1:], "\n")
	}
	return format, decklist
}

func callAnalyze(ctx context.Context, cfg *botConfig, decklist, format string) (*analyzeResponse, error) {
	payload, err := json.Marshal(analyzeRequest{Decklist: decklist, Format: format})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL+"/api/v1/analyze", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIJWT)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf(e.Error)
	}

	var out analyzeResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

func findCardByName(ctx context.Context, cfg *botConfig, name string) (*cardLookupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+"/api/v1/cards/search?name="+url.QueryEscape(name), nil)
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIJWT)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("card search request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf(e.Error)
	}
	var out cardLookupResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode card search: %w", err)
	}
	return &out, nil
}

func callPriceTrendByName(ctx context.Context, cfg *botConfig, cardName string) (*priceTrendResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+"/api/v1/cards/by-name/price-trend?name="+url.QueryEscape(cardName), nil)
	if err != nil {
		return nil, fmt.Errorf("build price-trend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIJWT)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("price-trend request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf(e.Error)
	}
	var out priceTrendResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode price trend: %w", err)
	}
	return &out, nil
}

func callSynergiesByName(ctx context.Context, cfg *botConfig, cardName string, n int) ([]synergyItem, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/cards/by-name/synergies?name=%s&n=%d", cfg.APIURL, url.QueryEscape(cardName), n), nil)
	if err != nil {
		return nil, "", fmt.Errorf("build synergies request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIJWT)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("synergies request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, "", fmt.Errorf(e.Error)
	}
	var out []synergyItem
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, "", fmt.Errorf("decode synergies: %w", err)
	}
	resolvedName := cardName
	if len(out) > 0 {
		resolvedName = titleCase(cardName)
	}
	return out, resolvedName, nil
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit-3] + "..."
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func formatPct(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.2f%%", *v)
}

func commandRemainder(content, command string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(command)) {
		return ""
	}
	return strings.TrimSpace(trimmed[len(command):])
}
