package urlcheck

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/rabbitjobs"
)

const defaultWorkers = 4

type Config struct {
	RabbitURL string
	Queue     string
	Workers   int
	Timeout   time.Duration
}

type jobConsumer interface {
	Jobs(ctx context.Context) <-chan rabbitjobs.URLCheckDelivery
	Close() error
}

func Run(ctx context.Context, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	cfg := LoadConfigFromEnv()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	consumer, err := rabbitjobs.NewConsumer(cfg.RabbitURL, cfg.Queue)
	if err != nil {
		return err
	}
	defer func() {
		if err = consumer.Close(); err != nil {
			logger.Error("close_rabbit_consumer_failed", slog.Any("error", err))
		}
	}()

	runWorkers(ctx, consumer, cfg, logger)
	return nil
}

func LoadConfigFromEnv() Config {
	workers := defaultWorkers
	if value := os.Getenv("URLCHECK_WORKERS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			workers = parsed
		}
	}

	timeout := 5 * time.Second
	if value := os.Getenv("URLCHECK_TIMEOUT"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	return Config{
		RabbitURL: getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		Queue:     getenv("RABBITMQ_URL_CHECK_QUEUE", "url.check"),
		Workers:   workers,
		Timeout:   timeout,
	}
}

func runWorkers(ctx context.Context, consumer jobConsumer, cfg Config, logger *slog.Logger) {
	client := &http.Client{Timeout: cfg.Timeout}
	jobs := consumer.Jobs(ctx)
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for delivery := range jobs {
				if err := checkURL(ctx, client, delivery.Job, logger); err != nil {
					_ = delivery.Nack(true)
					continue
				}
				_ = delivery.Ack()
			}
		}()
	}

	logger.Info("urlcheck_started", slog.Int("workers", cfg.Workers))
	<-ctx.Done()
	wg.Wait()
	logger.Info("urlcheck_stopped")
}

func checkURL(ctx context.Context, client *http.Client, job link.URLCheckJob, logger *slog.Logger) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, job.OriginalURL, nil)
	if err != nil {
		logger.Error("urlcheck_request_build_failed", slog.String("code", job.Code), slog.Any("error", err))
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		logger.Error("urlcheck_failed", slog.String("code", job.Code), slog.String("url", job.OriginalURL), slog.Any("error", err))
		return err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logger.Error("urlcheck_close_body_failed", slog.String("code", job.Code), slog.Any("error", err))
		}
	}()

	logger.Info("urlcheck_completed", slog.String("code", job.Code), slog.Int("status", resp.StatusCode))
	return nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
