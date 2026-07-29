package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr            = ":8080"
	defaultStartupTimeout  = 5 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultRedisCacheTTL   = 24 * time.Hour
	defaultGRPCAddr        = ":9090"
)

type Config struct {
	Addr            string
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	HTTP            HTTPConfig
	Database        DatabaseConfig
	Redis           RedisConfig
	GRPC            GRPCConfig
	Kafka           KafkaConfig
	RabbitMQ        RabbitMQConfig
}

type HTTPConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Addr      string
	Username  string
	Password  string
	DB        int
	KeyPrefix string
	CacheTTL  time.Duration
}

type GRPCConfig struct {
	Addr string
}

type KafkaConfig struct {
	Brokers    []string
	ClickTopic string
	Workers    int
	Buffer     int
}

type RabbitMQConfig struct {
	URL           string
	URLCheckQueue string
}

func LoadConfigFromEnv() (Config, error) {
	return loadConfig(os.LookupEnv)
}

func loadConfig(lookup func(string) (string, bool)) (Config, error) {
	redisDB, err := parseEnvInt(lookup, "REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	startupTimeout, err := parseEnvDuration(lookup, "SHORTENER_STARTUP_TIMEOUT", defaultStartupTimeout)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := parseEnvDuration(lookup, "SHORTENER_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := parseEnvDuration(lookup, "SHORTENER_READ_TIMEOUT", defaultReadTimeout)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := parseEnvDuration(lookup, "SHORTENER_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := parseEnvDuration(lookup, "SHORTENER_IDLE_TIMEOUT", defaultIdleTimeout)
	if err != nil {
		return Config{}, err
	}
	redisCacheTTL, err := parseEnvDuration(lookup, "REDIS_CACHE_TTL", defaultRedisCacheTTL)
	if err != nil {
		return Config{}, err
	}
	kafkaWorkers, err := parseEnvInt(lookup, "KAFKA_PUBLISH_WORKERS", 4)
	if err != nil {
		return Config{}, err
	}
	kafkaBuffer, err := parseEnvInt(lookup, "KAFKA_PUBLISH_BUFFER", 1024)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:            getenv(lookup, "SHORTENER_ADDR", defaultAddr),
		StartupTimeout:  startupTimeout,
		ShutdownTimeout: shutdownTimeout,
		HTTP: HTTPConfig{
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		Database: DatabaseConfig{
			URL: getenv(lookup, "DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Addr:      getenv(lookup, "REDIS_ADDR", ""),
			Username:  getenv(lookup, "REDIS_USERNAME", ""),
			Password:  getenv(lookup, "REDIS_PASSWORD", ""),
			DB:        redisDB,
			KeyPrefix: getenv(lookup, "REDIS_KEY_PREFIX", ""),
			CacheTTL:  redisCacheTTL,
		},
		GRPC: GRPCConfig{
			Addr: getenv(lookup, "SHORTENER_GRPC_ADDR", defaultGRPCAddr),
		},
		Kafka: KafkaConfig{
			Brokers:    splitCSV(getenv(lookup, "KAFKA_BROKERS", "")),
			ClickTopic: getenv(lookup, "KAFKA_CLICK_TOPIC", "link.clicked"),
			Workers:    kafkaWorkers,
			Buffer:     kafkaBuffer,
		},
		RabbitMQ: RabbitMQConfig{
			URL:           getenv(lookup, "RABBITMQ_URL", ""),
			URLCheckQueue: getenv(lookup, "RABBITMQ_URL_CHECK_QUEUE", "url.check"),
		},
	}, nil
}

func getenv(lookup func(string) (string, bool), key, fallback string) string {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func parseEnvInt(lookup func(string) (string, bool), key string, fallback int) (int, error) {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func parseEnvDuration(lookup func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
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
