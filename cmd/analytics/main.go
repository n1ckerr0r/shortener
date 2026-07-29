package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/n1ckerr0r/shortener/internal/analytics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := analytics.Run(context.Background(), logger); err != nil {
		logger.Error("analytics_stopped_with_error", slog.Any("error", err))
		os.Exit(1)
	}
}
