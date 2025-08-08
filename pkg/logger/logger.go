package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func GetDefaultLogger() *slog.Logger {
	w := os.Stderr

	return slog.New(
		tint.NewHandler(w, &tint.Options{
			AddSource:  true,
			Level:      getEnvLogLevel(slog.LevelInfo),
			TimeFormat: time.Kitchen,
			NoColor:    !isatty.IsTerminal(w.Fd()),
		}))
}

func getEnvLogLevel(defaultLevel slog.Level) slog.Level {
	// Read log level from environment variable
	level := defaultLevel
	if lvlStr := os.Getenv("LOG_LEVEL"); lvlStr != "" {
		if parsed, err := parseLogLevel(lvlStr); err == nil {
			level = parsed
		} else {
			slog.Warn("[slog] Invalid LOG_LEVEL env value, fallback to default. Supported levels (case-insensitive): debug, info, warn, error", "default", lvlStr)
		}
	}

	return level
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, os.ErrInvalid
	}
}
