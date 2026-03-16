package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/manawise/api/api/handlers"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// RouterDeps groups all handler dependencies.
type RouterDeps struct {
	CardRepo      domain.CardRepository
	UserRepo      domain.UserRepository
	DeckRepo      domain.DeckRepository
	AnalyzeUC     *usecase.AnalyzeDeckUseCase
	AISuggester   *usecase.AISuggester
	EmbedBatchUC  *usecase.EmbedBatchUseCase
	ResolveCardUC *usecase.ResolveCardByNameUseCase
	SideboardUC   *usecase.SideboardCoachUseCase
	MulliganUC    *usecase.MulliganAssistantUseCase
	MatchupUC     *usecase.MatchupSimulatorUseCase
	OTAUC         *usecase.OTAUpdateUseCase
	Analytics     domain.AnalyticsTracker
	JWTSecret     string
	ExpiryHours   int
}

// NewRouter builds and returns the chi router with all routes registered.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(corsMiddleware)

	// Instantiate handlers.
	authH := handlers.NewAuthHandler(deps.UserRepo, deps.JWTSecret, deps.ExpiryHours)
	analyzeH := handlers.NewAnalyzeHandler(deps.AnalyzeUC, deps.AISuggester, deps.UserRepo, deps.Analytics)
	cardsH := handlers.NewCardsHandler(deps.CardRepo, deps.ResolveCardUC)
	sideboardH := handlers.NewSideboardCoachHandler(deps.SideboardUC)
	mulliganH := handlers.NewMulliganHandler(deps.MulliganUC)
	matchupH := handlers.NewMatchupHandler(deps.MatchupUC)
	embedH := handlers.NewEmbedBatchHandler(deps.EmbedBatchUC)
	otaH := handlers.NewOTAHandler(deps.OTAUC)
	analyticsH := handlers.NewAnalyticsHandler(deps.Analytics)
	var deckH *handlers.DeckHandler
	if deps.DeckRepo != nil {
		deckH = handlers.NewDeckHandler(deps.DeckRepo)
	}

	jwtMW := middleware.JWTAuth(deps.JWTSecret)
	freemiumMW := middleware.FreemiumGate(deps.UserRepo, deps.Analytics)
	// 10 requests per minute per IP on auth endpoints (brute-force protection).
	authRateMW := middleware.AuthRateLimit(time.Minute, 10)

	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints.
		r.Get("/health", handlers.Health)

		// Auth endpoints — rate-limited per IP.
		r.Route("/auth", func(r chi.Router) {
			r.With(authRateMW).Post("/register", authH.Register)
			r.With(authRateMW).Post("/login", authH.Login)
			r.With(jwtMW).Get("/me", authH.Me)
		})

		// Protected endpoints.
		r.Group(func(r chi.Router) {
			r.Use(jwtMW)

			// Analyze — also gate by freemium quota.
			r.With(freemiumMW).Post("/analyze", analyzeH.ServeHTTP)
			r.Post("/sideboard/plan", sideboardH.ServeHTTP)
			r.Post("/mulligan/simulate", mulliganH.ServeHTTP)
			r.Post("/matchup/simulate", matchupH.ServeHTTP)

			// Cards.
			r.Get("/cards/search", cardsH.SearchByName)
			r.Get("/cards/by-name/price-trend", cardsH.PriceTrendByName)
			r.Get("/cards/by-name/synergies", cardsH.SynergiesByName)
			r.Get("/cards/{id}", cardsH.GetCard)
			r.Get("/cards/{id}/price-trend", cardsH.PriceTrend)
			r.Get("/cards/{id}/synergies", cardsH.GetSynergies)

			// Saved decks — only registered when DeckRepo is wired.
			if deckH != nil {
				r.Get("/decks", deckH.List)
				r.Post("/decks", deckH.Create)
				r.Get("/decks/{id}", deckH.Get)
				r.Put("/decks/{id}", deckH.Update)
				r.Delete("/decks/{id}", deckH.Delete)
			}

			// Embeddings pipeline.
			r.Post("/embed/batch", embedH.ServeHTTP)

			// Analytics.
			r.Post("/analytics/upgrade-click", analyticsH.UpgradeClick)

			// OTA (secure by JWT; in production restrict to admin/service accounts).
			r.Post("/ota/release", otaH.PublishRelease)
			r.Post("/ota/report-boot", otaH.ReportBoot)
			r.Get("/ota/manifest", otaH.Manifest)
		})
	})

	// SPA / frontend fallback — serves files from ./web/dist.
	fileServer := http.FileServer(http.Dir("./web/dist"))
	r.Handle("/*", fileServer)

	return r
}

// corsMiddleware sets permissive CORS headers for development.
// In production, restrict AllowedOrigins to the actual domain.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
