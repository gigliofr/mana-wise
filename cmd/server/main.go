package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/manawise/api/api"
	"github.com/manawise/api/config"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/infrastructure/analytics"
	"github.com/manawise/api/infrastructure/llm"
	"github.com/manawise/api/infrastructure/mongodb"
	"github.com/manawise/api/infrastructure/ota"
	"github.com/manawise/api/infrastructure/scryfall"
	"github.com/manawise/api/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	log.Printf("🔧 Environment: %s | Port: %s", cfg.Server.Environment, cfg.Server.Port)
	log.Printf("🔎 Mongo target: %s", sanitizeMongoURI(cfg.MongoDB.URI))

	// ── MongoDB ──────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	mongoClient, connectedAttempt, retryCount, err := connectMongoWithRetry(ctx, cfg.MongoDB, 3, 2*time.Second)
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			log.Printf("MongoDB disconnect error: %v", err)
		}
	}()
	log.Printf("✅ MongoDB connected (attempt=%d, retries=%d)", connectedAttempt, retryCount)

	// ── Repositories ─────────────────────────────────────────────────────────
	setupCtx := context.Background()

	cardRepo, err := mongodb.NewCardRepository(setupCtx, mongoClient)
	if err != nil {
		log.Fatalf("card repo: %v", err)
	}
	userRepo, err := mongodb.NewUserRepository(setupCtx, mongoClient)
	if err != nil {
		log.Fatalf("user repo: %v", err)
	}
	deckRepo, err := mongodb.NewDeckRepository(setupCtx, mongoClient)
	if err != nil {
		log.Fatalf("deck repo: %v", err)
	}
	log.Println("✅ Repositories ready")

	// ── Scryfall client ───────────────────────────────────────────────────────
	scryfallClient := scryfall.NewClient(
		cfg.Scryfall.BaseURL,
		cfg.Scryfall.Timeout,
		cfg.Scryfall.RateLimit,
	)

	// ── Use cases ────────────────────────────────────────────────────────────
	analyzeUC := usecase.NewAnalyzeDeckUseCase(scryfallClient, cardRepo, cfg.Worker.PoolSize)
	resolveCardUC := usecase.NewResolveCardByNameUseCase(scryfallClient, cardRepo)
	sideboardUC := usecase.NewSideboardCoachUseCase(cardRepo)
	mulliganUC := usecase.NewMulliganAssistantUseCase(cardRepo)
	matchupUC := usecase.NewMatchupSimulatorUseCase(cardRepo)
	var embedBatchUC *usecase.EmbedBatchUseCase
	var otaUC *usecase.OTAUpdateUseCase
	log.Printf("✅ Worker pool size: %d", cfg.Worker.PoolSize)

	// ── LLM connector (optional) ─────────────────────────────────────────────
	var primaryLLM *llm.Connector
	if cfg.LLM.APIKey != "" {
		primaryLLM = llm.NewConnector(
			cfg.LLM.Provider,
			cfg.LLM.APIKey,
			cfg.LLM.BaseURL,
			cfg.LLM.Model,
			cfg.LLM.MaxTokens,
			cfg.LLM.Timeout,
			cfg.LLM.CacheTTL,
		)
		log.Printf("✅ Primary LLM ready (provider: %s, model: %s)", cfg.LLM.Provider, cfg.LLM.Model)
	} else {
		if cfg.LLM.Provider == "gemini" {
			log.Println("⚠️  GEMINI_API_KEY not set — primary AI provider disabled")
		} else {
			log.Println("⚠️  OPENAI_API_KEY not set — primary AI provider disabled")
		}
	}

	var secondaryLLM *llm.Connector
	if cfg.LLM.SecondaryProvider != "" {
		if cfg.LLM.SecondaryAPIKey != "" {
			secondaryLLM = llm.NewConnector(
				cfg.LLM.SecondaryProvider,
				cfg.LLM.SecondaryAPIKey,
				cfg.LLM.SecondaryBaseURL,
				cfg.LLM.SecondaryModel,
				cfg.LLM.MaxTokens,
				cfg.LLM.Timeout,
				cfg.LLM.CacheTTL,
			)
			log.Printf("✅ Secondary LLM ready (provider: %s, model: %s)", cfg.LLM.SecondaryProvider, cfg.LLM.SecondaryModel)
		} else {
			log.Printf("⚠️  LLM secondary provider configured (%s) but API key is missing", cfg.LLM.SecondaryProvider)
		}
	}

	if primaryLLM != nil && cfg.LLM.Provider == "openai" {
		embedBatchUC = usecase.NewEmbedBatchUseCase(cardRepo, primaryLLM, cfg.Worker.PoolSize)
	} else if secondaryLLM != nil && cfg.LLM.SecondaryProvider == "openai" {
		embedBatchUC = usecase.NewEmbedBatchUseCase(cardRepo, secondaryLLM, cfg.Worker.PoolSize)
	}

	aiSuggester := usecase.NewAISuggester(cfg.LLM.AIMode, primaryLLM, secondaryLLM, cfg.LLM.InternalRulesEnabled)
	log.Printf("✅ AI suggester ready (mode: %s, internal_rules: %t)", cfg.LLM.AIMode, cfg.LLM.InternalRulesEnabled)

	// ── Analytics tracker (optional) ─────────────────────────────────────────
	var tracker domain.AnalyticsTracker = domain.NoopAnalyticsTracker{}
	if cfg.Analytics.Provider == "posthog" && cfg.Analytics.APIKey != "" {
		tracker = analytics.NewPostHogTracker(cfg.Analytics.APIKey, cfg.Analytics.Host)
		log.Println("✅ Analytics tracker enabled (PostHog)")
	} else {
		log.Println("ℹ️  Analytics tracker disabled")
	}

	// ── OTA storage/use case ────────────────────────────────────────────────
	otaRepo, err := ota.NewStorageRepository(cfg.OTA.StorageDir)
	if err != nil {
		log.Fatalf("ota storage: %v", err)
	}
	otaUC = usecase.NewOTAUpdateUseCase(otaRepo)

	// ── Router ───────────────────────────────────────────────────────────────
	router := api.NewRouter(api.RouterDeps{
		CardRepo:      cardRepo,
		UserRepo:      userRepo,
		DeckRepo:      deckRepo,
		AnalyzeUC:     analyzeUC,
		SideboardUC:   sideboardUC,
		MulliganUC:    mulliganUC,
		MatchupUC:     matchupUC,
		AISuggester:   aiSuggester,
		EmbedBatchUC:  embedBatchUC,
		ResolveCardUC: resolveCardUC,
		OTAUC:         otaUC,
		Analytics:     tracker,
		JWTSecret:     cfg.JWT.Secret,
		ExpiryHours:   cfg.JWT.ExpiryHours,
	})

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 45 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 ManaWise API listening on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("🛑 Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("✅ Server stopped cleanly")
}

func connectMongoWithRetry(ctx context.Context, cfg config.MongoDBConfig, attempts int, baseBackoff time.Duration) (*mongodb.Client, int, int, error) {
	var lastErr error
	if attempts < 1 {
		attempts = 1
	}
	for i := 1; i <= attempts; i++ {
		client, err := mongodb.NewClient(ctx, cfg)
		if err == nil {
			return client, i, i - 1, nil
		}
		lastErr = err
		if i == attempts {
			break
		}
		backoff := time.Duration(i) * baseBackoff
		log.Printf("⚠️  MongoDB connect attempt %d/%d failed: %v | retry in %s", i, attempts, err, backoff)
		select {
		case <-ctx.Done():
			return nil, 0, i - 1, fmt.Errorf("mongodb connect aborted by context: %w", ctx.Err())
		case <-time.After(backoff):
		}
	}
	return nil, 0, attempts - 1, fmt.Errorf("mongodb connect failed after %d attempts: %w", attempts, lastErr)
}

func sanitizeMongoURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "<empty>"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid-uri>"
	}
	host := u.Hostname()
	if host == "" {
		return "<unknown-host>"
	}
	port := u.Port()
	if port != "" {
		return host + ":" + port
	}
	return host
}
