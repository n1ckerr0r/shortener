package analytics

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/n1ckerr0r/shortener/internal/analytics/mongostore"
	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/kafkaevents"
)

const (
	defaultKafkaTopic       = "link.clicked"
	defaultKafkaGroupID     = "shortener-analytics"
	defaultMongoDatabase    = "shortener"
	defaultMongoCollection  = "clicks"
	defaultAnalyticsWorkers = 4
)

type Config struct {
	KafkaBrokers    []string
	KafkaTopic      string
	KafkaGroupID    string
	MongoURI        string
	MongoDatabase   string
	MongoCollection string
	Workers         int
}

type clickStore interface {
	SaveClick(ctx context.Context, event link.ClickEvent) error
}

type clickConsumer interface {
	ReadClick(ctx context.Context) (link.ClickEvent, error)
	Close() error
}

func Run(ctx context.Context, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	cfg := LoadConfigFromEnv()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupCtx, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStartup()

	store, err := mongostore.Open(startupCtx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoCollection)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err = store.Close(shutdownCtx); err != nil {
			logger.Error("close_mongo_failed", slog.Any("error", err))
		}
	}()

	consumer := kafkaevents.NewConsumer(kafkaevents.ConsumerOptions{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopic,
		GroupID: cfg.KafkaGroupID,
	})
	defer func() {
		if err = consumer.Close(); err != nil {
			logger.Error("close_kafka_consumer_failed", slog.Any("error", err))
		}
	}()

	runWorkers(ctx, consumer, store, cfg.Workers, logger)
	return nil
}

func LoadConfigFromEnv() Config {
	workers := defaultAnalyticsWorkers
	if value := os.Getenv("ANALYTICS_WORKERS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			workers = parsed
		}
	}

	return Config{
		KafkaBrokers:    splitCSV(getenv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:      getenv("KAFKA_CLICK_TOPIC", defaultKafkaTopic),
		KafkaGroupID:    getenv("KAFKA_GROUP_ID", defaultKafkaGroupID),
		MongoURI:        getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:   getenv("MONGO_DATABASE", defaultMongoDatabase),
		MongoCollection: getenv("MONGO_CLICK_COLLECTION", defaultMongoCollection),
		Workers:         workers,
	}
}

func runWorkers(ctx context.Context, consumer clickConsumer, store clickStore, workers int, logger *slog.Logger) {
	events := make(chan link.ClickEvent, workers*2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(events)
		for {
			event, err := consumer.ReadClick(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
					return
				}
				logger.Error("read_click_event_failed", slog.Any("error", err))
				continue
			}

			select {
			case <-ctx.Done():
				return
			case events <- event:
			}
		}
	}()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range events {
				if err := store.SaveClick(ctx, event); err != nil {
					logger.Error("save_click_event_failed", slog.String("code", event.Code), slog.Any("error", err))
				}
			}
		}()
	}

	logger.Info("analytics_started", slog.Int("workers", workers))
	<-ctx.Done()
	wg.Wait()
	logger.Info("analytics_stopped")
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}

	return values
}
