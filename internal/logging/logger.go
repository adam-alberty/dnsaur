package logging

import (
	"log/slog"
	"os"

	"github.com/adam-alberty/dnsaur/internal/config"
)

// Sets up logger from configuration.
func SetupLogger(cfg config.LoggingConfig) {
	level := slog.LevelInfo

	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	loggerOptions := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, loggerOptions)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, loggerOptions)
	default:
		handler = slog.NewTextHandler(os.Stdout, loggerOptions)
	}

	slog.SetDefault(slog.New(handler))
}
