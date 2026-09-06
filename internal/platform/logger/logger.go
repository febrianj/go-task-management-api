package logger

import (
	"log/slog"
	"os"
	"strings"
)

func ParseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func New(level string, isProduction bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     ParseLevel(level),
		AddSource: !isProduction,
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
