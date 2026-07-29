package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/kafkaevents"
	"github.com/n1ckerr0r/shortener/internal/link/memory"
	"github.com/n1ckerr0r/shortener/internal/link/postgres"
	"github.com/n1ckerr0r/shortener/internal/link/rabbitjobs"
	"github.com/n1ckerr0r/shortener/internal/link/rediscache"
)

type repositoryHandle struct {
	repository  link.Repository
	closeFn     func() error
	readyChecks []ReadyCheck
}

type cacheHandle struct {
	cache       link.ResolveCache
	ttl         time.Duration
	closeFn     func() error
	readyChecks []ReadyCheck
}

type publisherHandle struct {
	clickPublisher    link.ClickPublisher
	urlCheckPublisher link.URLCheckPublisher
	closeFns          []func() error
}

func buildRepository(ctx context.Context, cfg Config, logger *slog.Logger) (repositoryHandle, error) {
	if cfg.Database.URL == "" {
		logger.Info("storage_configured", slog.String("kind", "memory"))
		return repositoryHandle{
			repository: memory.NewRepository(),
			closeFn:    func() error { return nil },
		}, nil
	}

	repo, err := postgres.Open(ctx, cfg.Database.URL)
	if err != nil {
		return repositoryHandle{}, fmt.Errorf("connect postgres: %w", err)
	}

	logger.Info("storage_configured", slog.String("kind", "postgres"))
	return repositoryHandle{
		repository: repo,
		closeFn:    repo.Close,
		readyChecks: []ReadyCheck{
			{
				Name:  "postgres",
				Check: repo.Ping,
			},
		},
	}, nil
}

func buildResolveCache(ctx context.Context, cfg Config, logger *slog.Logger) (cacheHandle, error) {
	if cfg.Redis.Addr == "" {
		logger.Info("resolve_cache_configured", slog.String("kind", "disabled"))
		return cacheHandle{
			closeFn: func() error { return nil },
		}, nil
	}

	cache, err := rediscache.Open(ctx, rediscache.Options{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Prefix:   cfg.Redis.KeyPrefix,
	})
	if err != nil {
		return cacheHandle{}, fmt.Errorf("connect redis: %w", err)
	}

	logger.Info("resolve_cache_configured", slog.String("kind", "redis"), slog.String("addr", cfg.Redis.Addr))
	return cacheHandle{
		cache:   cache,
		ttl:     cfg.Redis.CacheTTL,
		closeFn: cache.Close,
		readyChecks: []ReadyCheck{
			{
				Name:  "redis",
				Check: cache.Ping,
			},
		},
	}, nil
}

func buildPublishers(cfg Config, logger *slog.Logger) (publisherHandle, error) {
	var handle publisherHandle

	if len(cfg.Kafka.Brokers) > 0 {
		publisher := kafkaevents.NewPublisher(kafkaevents.PublisherOptions{
			Brokers: cfg.Kafka.Brokers,
			Topic:   cfg.Kafka.ClickTopic,
			Workers: cfg.Kafka.Workers,
			Buffer:  cfg.Kafka.Buffer,
			Logger:  logger,
		})
		handle.clickPublisher = publisher
		handle.closeFns = append(handle.closeFns, publisher.Close)
		logger.Info("kafka_click_publisher_configured", slog.Any("brokers", cfg.Kafka.Brokers), slog.String("topic", cfg.Kafka.ClickTopic))
	}

	if cfg.RabbitMQ.URL != "" {
		publisher, err := rabbitjobs.NewPublisher(cfg.RabbitMQ.URL, cfg.RabbitMQ.URLCheckQueue)
		if err != nil {
			return publisherHandle{}, fmt.Errorf("connect rabbitmq: %w", err)
		}
		handle.urlCheckPublisher = publisher
		handle.closeFns = append(handle.closeFns, publisher.Close)
		logger.Info("rabbitmq_urlcheck_publisher_configured", slog.String("queue", cfg.RabbitMQ.URLCheckQueue))
	}

	return handle, nil
}

func (h repositoryHandle) close(logger *slog.Logger) {
	if h.closeFn == nil {
		return
	}
	if err := h.closeFn(); err != nil {
		logger.Error("close_repository_failed", slog.Any("error", err))
	}
}

func (h cacheHandle) close(logger *slog.Logger) {
	if h.closeFn == nil {
		return
	}
	if err := h.closeFn(); err != nil {
		logger.Error("close_cache_failed", slog.Any("error", err))
	}
}

func (h publisherHandle) close(logger *slog.Logger) {
	for _, closeFn := range h.closeFns {
		if err := closeFn(); err != nil {
			logger.Error("close_publisher_failed", slog.Any("error", err))
		}
	}
}
