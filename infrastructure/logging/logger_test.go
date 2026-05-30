package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestInitLogger_Development(t *testing.T) {
	logger := InitLogger("DEBUG", "development")
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestInitLogger_Production(t *testing.T) {
	logger := InitLogger("INFO", "production")
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestInitLoggerWithWriter_JSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := InitLoggerWithWriter("INFO", "production", buf)

	logger.Info("test message", slog.String("key", "value"))

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Fatalf("expected 'test message' in output, got: %s", output)
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		level    string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo}, // default
	}

	for _, c := range cases {
		got := parseLogLevel(c.level)
		if got != c.expected {
			t.Fatalf("expected %v for %s, got %v", c.expected, c.level, got)
		}
	}
}
