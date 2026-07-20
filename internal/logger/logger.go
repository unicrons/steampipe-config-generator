// Package logger builds the slog.Logger used by the CLI. It is only imported by package cmd -
// generator and internal/aws never log, they return errors instead.
package logger

import (
	"log/slog"
	"os"
)

// New returns a logger writing to stderr: human-readable text for "default", structured JSON
// for "json". Its level is read from the LOG_LEVEL env var (default info).
func New(format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: levelFromEnv()}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

func levelFromEnv() slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		return slog.LevelInfo
	}
	return level
}
