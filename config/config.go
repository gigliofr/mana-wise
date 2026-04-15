package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server    ServerConfig
	MongoDB   MongoDBConfig
	JWT       JWTConfig
	Scryfall  ScryfallConfig
	LLM       LLMConfig
	Worker    WorkerConfig
	Analytics AnalyticsConfig
	OTA       OTAConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port        string
	Environment string
	LogLevel    string
}

// MongoDBConfig contains MongoDB connection settings.
type MongoDBConfig struct {
	URI         string
	Database    string
	TLSCertFile string // path to PEM file containing client cert + private key (for X.509 auth)
}

// JWTConfig contains JWT signing settings.
type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// ScryfallConfig contains Scryfall API settings.
type ScryfallConfig struct {
	BaseURL   string
	Timeout   time.Duration
	RateLimit int // requests per second
}

// LLMConfig contains LLM API settings.
type LLMConfig struct {
	Provider             string // "openai" | "gemini"
	APIKey               string
	BaseURL              string
	Model                string
	SecondaryProvider    string
	SecondaryAPIKey      string
	SecondaryBaseURL     string
	SecondaryModel       string
	AIMode               string // "external_only" | "internal_only" | "hybrid_prefer_external" | "hybrid_prefer_internal"
	InternalRulesEnabled bool
	FallbackOnStatus     []int
	FallbackOnTimeout    bool
	MaxTokens            int
	Timeout              time.Duration
	CacheTTL             time.Duration
}

// WorkerConfig contains worker pool settings.
type WorkerConfig struct {
	PoolSize int
}

// AnalyticsConfig contains product analytics settings.
type AnalyticsConfig struct {
	Provider string // "none" | "posthog"
	APIKey   string
	Host     string
}

// OTAConfig contains secure OTA storage configuration.
type OTAConfig struct {
	StorageDir string
}

// Load reads configuration from environment variables and returns a Config.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{}

	// Server
	cfg.Server.Port = getEnv("PORT", "8080")
	cfg.Server.Environment = getEnv("ENVIRONMENT", "development")
	cfg.Server.LogLevel = getEnv("LOG_LEVEL", "INFO")

	// MongoDB
	cfg.MongoDB.URI = firstNonEmptyEnv("MONGODB_URI", "DATABASE_URL", "MONGO_URL", "MONGODB_URL")
	if cfg.MongoDB.URI == "" {
		return nil, fmt.Errorf("MongoDB URI is required: set one of MONGODB_URI, DATABASE_URL, MONGO_URL, MONGODB_URL")
	}
	cfg.MongoDB.Database = getEnv("MONGODB_DB_NAME", "manawise")
	cfg.MongoDB.TLSCertFile = strings.TrimSpace(os.Getenv("MONGODB_TLS_CERT_FILE"))

	// JWT
	cfg.JWT.Secret = strings.TrimSpace(firstNonEmptyEnv("JWT_SECRET", "SECRET", "APP_SECRET"))
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT signing secret is required: set one of JWT_SECRET, SECRET, APP_SECRET")
	}
	expiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "72"))
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRY_HOURS must be a valid integer: %w", err)
	}
	cfg.JWT.ExpiryHours = expiry

	// Scryfall
	cfg.Scryfall.BaseURL = getEnv("SCRYFALL_BASE_URL", "https://api.scryfall.com")
	cfg.Scryfall.Timeout = 5 * time.Second
	cfg.Scryfall.RateLimit = 10

	// LLM primary provider
	cfg.LLM.Provider = normalizeProvider(getEnv("LLM_PROVIDER", "openai"))
	switch cfg.LLM.Provider {
	case "gemini":
		cfg.LLM.APIKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
		cfg.LLM.BaseURL = getEnv("GEMINI_OPENAI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai")
		cfg.LLM.Model = getEnv("LLM_MODEL", "gemini-1.5-pro")
	default:
		cfg.LLM.Provider = "openai"
		cfg.LLM.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		cfg.LLM.BaseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
		cfg.LLM.Model = getEnv("LLM_MODEL", "gpt-4o-mini")
	}

	// LLM secondary provider (optional)
	secondaryProviderRaw := strings.TrimSpace(getEnv("LLM_SECONDARY_PROVIDER", ""))
	if secondaryProviderRaw != "" {
		cfg.LLM.SecondaryProvider = normalizeProvider(secondaryProviderRaw)
	}
	if cfg.LLM.SecondaryProvider != "" {
		switch cfg.LLM.SecondaryProvider {
		case "gemini":
			cfg.LLM.SecondaryAPIKey = strings.TrimSpace(os.Getenv("LLM_SECONDARY_API_KEY"))
			if cfg.LLM.SecondaryAPIKey == "" {
				cfg.LLM.SecondaryAPIKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
			}
			cfg.LLM.SecondaryBaseURL = strings.TrimSpace(getEnv("LLM_SECONDARY_BASE_URL", getEnv("GEMINI_OPENAI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai")))
			cfg.LLM.SecondaryModel = strings.TrimSpace(getEnv("LLM_SECONDARY_MODEL", "gemini-pro-latest"))
		default:
			cfg.LLM.SecondaryProvider = "openai"
			cfg.LLM.SecondaryAPIKey = strings.TrimSpace(os.Getenv("LLM_SECONDARY_API_KEY"))
			if cfg.LLM.SecondaryAPIKey == "" {
				cfg.LLM.SecondaryAPIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
			}
			cfg.LLM.SecondaryBaseURL = strings.TrimSpace(getEnv("LLM_SECONDARY_BASE_URL", strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))))
			cfg.LLM.SecondaryModel = strings.TrimSpace(getEnv("LLM_SECONDARY_MODEL", getEnv("LLM_MODEL", "gpt-4o-mini")))
		}
	}

	cfg.LLM.AIMode = normalizeAIMode(getEnv("AI_MODE", "hybrid_prefer_external"))
	internalEnabled, err := strconv.ParseBool(getEnv("AI_INTERNAL_RULES_ENABLED", "true"))
	if err != nil {
		return nil, fmt.Errorf("AI_INTERNAL_RULES_ENABLED must be true or false: %w", err)
	}
	cfg.LLM.InternalRulesEnabled = internalEnabled

	fallbackStatuses, err := parseStatusCodeList(getEnv("AI_FALLBACK_ON_STATUS", "429,500,502,503,504"))
	if err != nil {
		return nil, fmt.Errorf("AI_FALLBACK_ON_STATUS must be a comma-separated list of valid HTTP status codes: %w", err)
	}
	cfg.LLM.FallbackOnStatus = fallbackStatuses

	timeoutMsRaw := strings.TrimSpace(os.Getenv("AI_FALLBACK_ON_TIMEOUT_MS"))
	cfg.LLM.Timeout = 15 * time.Second
	cfg.LLM.FallbackOnTimeout = true
	if timeoutMsRaw != "" {
		timeoutMs, parseErr := strconv.Atoi(timeoutMsRaw)
		if parseErr != nil {
			return nil, fmt.Errorf("AI_FALLBACK_ON_TIMEOUT_MS must be a valid integer (milliseconds): %w", parseErr)
		}
		if timeoutMs <= 0 {
			cfg.LLM.FallbackOnTimeout = false
		} else {
			cfg.LLM.Timeout = time.Duration(timeoutMs) * time.Millisecond
		}
	}

	maxTokens, err := strconv.Atoi(getEnv("LLM_MAX_TOKENS", "1000"))
	if err != nil {
		return nil, fmt.Errorf("LLM_MAX_TOKENS must be a valid integer: %w", err)
	}
	cfg.LLM.MaxTokens = maxTokens
	cfg.LLM.CacheTTL = 1 * time.Hour

	// Worker
	poolSize, err := strconv.Atoi(getEnv("WORKER_POOL_SIZE", "20"))
	if err != nil {
		return nil, fmt.Errorf("WORKER_POOL_SIZE must be a valid integer: %w", err)
	}
	cfg.Worker.PoolSize = poolSize

	// Analytics
	cfg.Analytics.Provider = getEnv("ANALYTICS_PROVIDER", "none")
	cfg.Analytics.APIKey = strings.TrimSpace(os.Getenv("POSTHOG_API_KEY"))
	cfg.Analytics.Host = getEnv("POSTHOG_HOST", "https://app.posthog.com")

	// OTA
	cfg.OTA.StorageDir = getEnv("OTA_STORAGE_DIR", "./ota-releases")

	return cfg, nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "", "openai", "openai_compatible", "openrouter", "groq", "together":
		return "openai"
	case "gemini":
		return "gemini"
	default:
		return "openai"
	}
}

func normalizeAIMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "external_only", "internal_only", "hybrid_prefer_external", "hybrid_prefer_internal":
		return mode
	default:
		return "hybrid_prefer_external"
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func parseStatusCodeList(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}, nil
	}

	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		code, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		if code < 100 || code > 599 {
			return nil, fmt.Errorf("status %d out of HTTP range", code)
		}
		out = append(out, code)
	}
	return out, nil
}
