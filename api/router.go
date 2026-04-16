package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gigliofr/mana-wise/api/handlers"
	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

// RouterDeps groups all handler dependencies.
type RouterDeps struct {
	CardRepo         domain.CardRepository
	UserRepo         domain.UserRepository
	DeckRepo         domain.DeckRepository
	AnalyzeUC        *usecase.AnalyzeDeckUseCase
	AISuggester      *usecase.AISuggester
	EmbedBatchUC     *usecase.EmbedBatchUseCase
	ResolveCardUC    *usecase.ResolveCardByNameUseCase
	SideboardUC      *usecase.SideboardCoachUseCase
	MulliganUC       *usecase.MulliganAssistantUseCase
	MatchupUC        *usecase.MatchupSimulatorUseCase
	DeckClassifyUC   *usecase.DeckClassifierUseCase
	OTAUC            *usecase.OTAUpdateUseCase
	ScoreUC          *usecase.ScoreUseCase
	ImpactScoreUC    *usecase.ImpactScoreUseCase
	Analytics        domain.AnalyticsTracker
	AnalyticsMetrics domain.AnalyticsMetricsProvider
	PasswordResetRepo domain.PasswordResetTokenRepository
	Mailer            domain.EmailSender
	JWTSecret        string
	ExpiryHours      int
}

// NewRouter builds and returns the chi router with all routes registered.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(corsMiddleware)

	// Instantiate handlers.
	authH := handlers.NewAuthHandler(deps.UserRepo, deps.JWTSecret, deps.ExpiryHours).
		WithPasswordResetRepo(deps.PasswordResetRepo).
		WithMailer(deps.Mailer)
	analyzeH := handlers.NewAnalyzeHandler(deps.AnalyzeUC, deps.AISuggester, deps.UserRepo, deps.Analytics)
	cardsH := handlers.NewCardsHandler(deps.CardRepo, deps.ResolveCardUC)
	sideboardH := handlers.NewSideboardCoachHandler(deps.SideboardUC)
	mulliganH := handlers.NewMulliganHandler(deps.MulliganUC)
	matchupH := handlers.NewMatchupHandler(deps.MatchupUC)
	deckClassifyH := handlers.NewDeckClassifyHandler(deps.DeckClassifyUC)
	embedH := handlers.NewEmbedBatchHandler(deps.EmbedBatchUC)
	otaH := handlers.NewOTAHandler(deps.OTAUC)
	analyticsH := handlers.NewAnalyticsHandler(deps.Analytics)
	adminH := handlers.NewAdminHandler(deps.UserRepo, deps.AnalyticsMetrics)
	scoreH := handlers.NewScoreHandler(deps.AnalyzeUC, deps.ScoreUC, deps.ImpactScoreUC, deps.UserRepo)
	metaH := handlers.NewMetaHandler()
	notificationH := handlers.NewNotificationHandler(deps.DeckRepo, deps.CardRepo)
	var deckH *handlers.DeckHandler
	var deckImportExportH *handlers.DeckImportExportHandler
	if deps.DeckRepo != nil {
		deckH = handlers.NewDeckHandler(deps.DeckRepo, deps.UserRepo, deps.CardRepo, deps.AnalyzeUC, deps.DeckClassifyUC, deps.MulliganUC).WithTracker(deps.Analytics)
		deckImportExportH = handlers.NewDeckImportExportHandler(deps.DeckRepo, deps.UserRepo, deps.CardRepo, deps.ResolveCardUC)
	}

	jwtMW := middleware.JWTAuth(deps.JWTSecret)
	freemiumMW := middleware.FreemiumGate(deps.UserRepo, deps.Analytics)
	// 10 requests per minute per IP on auth endpoints (brute-force protection).
	authRateMW := middleware.AuthRateLimit(time.Minute, 10)

	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints.
		r.Get("/health", handlers.Health)
		r.Get("/meta/{format}", metaH.Snapshot)
		r.Post("/webhooks/scryfall", notificationH.IngestScryfallWebhook)

		// Auth endpoints — rate-limited per IP.
		r.Route("/auth", func(r chi.Router) {
			r.With(authRateMW).Post("/register", authH.Register)
			r.With(authRateMW).Post("/login", authH.Login)
			r.With(authRateMW).Post("/forgot-password", authH.ForgotPassword)
			r.With(authRateMW).Post("/reset-password", authH.ResetPassword)
			r.With(jwtMW).Post("/refresh", authH.Refresh)
			r.With(jwtMW).Get("/me", authH.Me)
			r.With(jwtMW).Post("/plan", authH.UpdatePlan)
		})

		// Protected endpoints.
		r.Group(func(r chi.Router) {
			r.Use(jwtMW)

			// Analyze — also gate by freemium quota.
			r.With(freemiumMW).Post("/analyze", analyzeH.ServeHTTP)
			r.With(freemiumMW).Post("/score", scoreH.Score)
			r.Get("/users/me/notifications", notificationH.Feed)
			r.Post("/sideboard/plan", sideboardH.ServeHTTP)
			r.Post("/mulligan/simulate", mulliganH.ServeHTTP)
			r.Post("/matchup/simulate", matchupH.ServeHTTP)
			r.Post("/deck/classify", deckClassifyH.ServeHTTP)

			// Cards.
			r.Get("/cards/search", cardsH.SearchByName)
			r.Post("/cards/metadata/batch", cardsH.MetadataBatch)
			r.Get("/cards/by-name/price-trend", cardsH.PriceTrendByName)
			r.Get("/cards/by-name/synergies", cardsH.SynergiesByName)
			r.Get("/cards/{id}", cardsH.GetCard)
			r.Get("/cards/{id}/price-trend", cardsH.PriceTrend)
			r.Get("/cards/{id}/synergies", cardsH.GetSynergies)

			// Saved decks — only registered when DeckRepo is wired.
			if deckH != nil {
				r.Get("/users/me/collection/gaps/{deck_id}", deckH.CollectionGaps)
				r.Post("/decks/import", deckImportExportH.Import)
				r.Get("/decks", deckH.List)
				r.Post("/decks", deckH.Create)
				r.Get("/decks/{id}", deckH.Get)
				r.Get("/decks/{id}/summary", deckH.Summary)
				r.Get("/decks/{id}/price", deckH.Price)
				r.Get("/decks/{id}/budget", deckH.Budget)
				r.Get("/decks/{id}/analysis", deckH.Analysis)
				r.Get("/decks/{id}/synergies", deckH.Synergies)
				r.Post("/decks/{id}/sideboard/suggest", deckH.SideboardSuggest)
				r.Post("/decks/{id}/simulate", deckH.Simulate)
				r.Get("/decks/{id}/history", deckH.History)
				r.Post("/decks/{id}/restore/{version}", deckH.Restore)
				r.Get("/decks/{id}/legality", deckH.Legality)
				r.Get("/decks/{id}/export", deckImportExportH.Export)
				r.Put("/decks/{id}", deckH.Update)
				r.Delete("/decks/{id}", deckH.Delete)
			}

			// Embeddings pipeline.
			r.Post("/embed/batch", embedH.ServeHTTP)

			// Analytics.
			r.Post("/analytics/upgrade-click", analyticsH.UpgradeClick)

			// OTA routes can be disabled in non-firmware deployments.
			if otaEnabledFromEnv() {
				r.Post("/ota/release", otaH.PublishRelease)
				r.Post("/ota/report-boot", otaH.ReportBoot)
				r.Get("/ota/manifest", otaH.Manifest)
			}
		})

		// Admin endpoints — protected by ADMIN_SECRET header.
		r.Route("/admin", func(r chi.Router) {
			r.Use(handlers.AdminSecretMiddleware)
			r.Post("/user/plan", adminH.UpdateUserPlan)
			r.Get("/metrics/funnel", adminH.FunnelMetrics)
		})
	})

	// SPA / frontend fallback.
	// Serve static assets directly, and fallback to index.html for client routes
	// (e.g. /login) so deep links in the React app don't return 404.
	r.Handle("/*", spaFallbackHandler("./web/dist"))

	return r
}

func spaFallbackHandler(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))
	indexFile := filepath.Join(distDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." || cleanPath == string(filepath.Separator) {
			http.ServeFile(w, r, indexFile)
			return
		}

		candidate := filepath.Join(distDir, cleanPath)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexFile)
	})
}

// corsMiddleware sets permissive CORS headers for development.
// In production, restrict AllowedOrigins to the actual domain.
func corsMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		})
	}

	allowedOrigins := allowedOriginsFromEnv()
	allowAny := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowedSet := map[string]bool{}
	for _, origin := range allowedOrigins {
		allowedSet[origin] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic in corsMiddleware: %v\\n%s", rec, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowOrigin := ""
		if allowAny {
			allowOrigin = "*"
		} else if origin != "" && allowedSet[origin] {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if origin != "" && allowOrigin == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"origin not allowed"}`))
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("MANAWISE_ALLOWED_ORIGINS"))
	if raw == "" {
		env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
		if env == "" || env == "development" || env == "dev" {
			return []string{"*"}
		}
		// Production-safe default: no cross-origin access unless explicitly allowlisted.
		return []string{}
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		origin := strings.TrimSpace(p)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return []string{"*"}
		}
		out = append(out, origin)
	}
	return out
}

func otaEnabledFromEnv() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("OTA_ENABLED")))
	if raw == "" {
		// Keep current behavior unless explicitly disabled.
		return true
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}
