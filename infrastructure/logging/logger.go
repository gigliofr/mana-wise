package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// InitLogger initializes a structured logger based on environment.
// logLevel: DEBUG, INFO, WARN, ERROR (default: INFO)
// environment: development, production (affects output format)
func InitLogger(logLevel, environment string) *slog.Logger {
	level := parseLogLevel(logLevel)
	
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(environment) == "production" {
		// JSON output in production
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Human-readable text output in development
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// InitLoggerWithWriter creates a logger that writes to the given writer.
func InitLoggerWithWriter(logLevel, environment string, w io.Writer) *slog.Logger {
	level := parseLogLevel(logLevel)
	
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(environment) == "production" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
