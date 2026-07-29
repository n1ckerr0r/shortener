package app

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	cfg, err := loadConfig(emptyEnv)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
	if cfg.StartupTimeout != defaultStartupTimeout {
		t.Fatalf("expected default startup timeout %s, got %s", defaultStartupTimeout, cfg.StartupTimeout)
	}
	if cfg.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("expected default shutdown timeout %s, got %s", defaultShutdownTimeout, cfg.ShutdownTimeout)
	}
	if cfg.HTTP.ReadTimeout != defaultReadTimeout {
		t.Fatalf("expected default read timeout %s, got %s", defaultReadTimeout, cfg.HTTP.ReadTimeout)
	}
	if cfg.Redis.CacheTTL != defaultRedisCacheTTL {
		t.Fatalf("expected default redis cache ttl %s, got %s", defaultRedisCacheTTL, cfg.Redis.CacheTTL)
	}
}

func TestLoadConfigReadsEnvValues(t *testing.T) {
	env := map[string]string{
		"SHORTENER_ADDR":             ":9090",
		"SHORTENER_STARTUP_TIMEOUT":  "3s",
		"SHORTENER_SHUTDOWN_TIMEOUT": "7s",
		"SHORTENER_READ_TIMEOUT":     "2s",
		"SHORTENER_WRITE_TIMEOUT":    "4s",
		"SHORTENER_IDLE_TIMEOUT":     "30s",
		"DATABASE_URL":               "postgres://user:pass@localhost/db?sslmode=disable",
		"REDIS_ADDR":                 "localhost:6379",
		"REDIS_USERNAME":             "default",
		"REDIS_PASSWORD":             "secret",
		"REDIS_DB":                   "2",
		"REDIS_KEY_PREFIX":           "test:",
		"REDIS_CACHE_TTL":            "15m",
	}

	cfg, err := loadConfig(mapEnv(env))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Addr != ":9090" {
		t.Fatalf("expected addr :9090, got %q", cfg.Addr)
	}
	if cfg.StartupTimeout != 3*time.Second {
		t.Fatalf("expected startup timeout 3s, got %s", cfg.StartupTimeout)
	}
	if cfg.ShutdownTimeout != 7*time.Second {
		t.Fatalf("expected shutdown timeout 7s, got %s", cfg.ShutdownTimeout)
	}
	if cfg.HTTP.ReadTimeout != 2*time.Second {
		t.Fatalf("expected read timeout 2s, got %s", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 4*time.Second {
		t.Fatalf("expected write timeout 4s, got %s", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout 30s, got %s", cfg.HTTP.IdleTimeout)
	}
	if cfg.Database.URL != env["DATABASE_URL"] {
		t.Fatalf("expected database url to be read")
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Fatalf("expected redis addr localhost:6379, got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.Username != "default" {
		t.Fatalf("expected redis username default, got %q", cfg.Redis.Username)
	}
	if cfg.Redis.Password != "secret" {
		t.Fatalf("expected redis password secret, got %q", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("expected redis db 2, got %d", cfg.Redis.DB)
	}
	if cfg.Redis.KeyPrefix != "test:" {
		t.Fatalf("expected redis key prefix test:, got %q", cfg.Redis.KeyPrefix)
	}
	if cfg.Redis.CacheTTL != 15*time.Minute {
		t.Fatalf("expected redis cache ttl 15m, got %s", cfg.Redis.CacheTTL)
	}
}

func TestLoadConfigRejectsInvalidRedisDB(t *testing.T) {
	_, err := loadConfig(mapEnv(map[string]string{
		"REDIS_DB": "invalid",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigRejectsInvalidDuration(t *testing.T) {
	_, err := loadConfig(mapEnv(map[string]string{
		"SHORTENER_SHUTDOWN_TIMEOUT": "invalid",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func emptyEnv(string) (string, bool) {
	return "", false
}

func mapEnv(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
