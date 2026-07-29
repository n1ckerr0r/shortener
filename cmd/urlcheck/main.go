package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/n1ckerr0r/shortener/internal/urlcheck"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := urlcheck.Run(context.Background(), logger); err != nil {
		logger.Error("urlcheck_stopped_with_error", slog.Any("error", err))
		os.Exit(1)
	}
}
