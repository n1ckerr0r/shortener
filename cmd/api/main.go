package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/httpapi"
	"github.com/n1ckerr0r/shortener/internal/link/memory"
	"github.com/n1ckerr0r/shortener/internal/link/postgres"
	"github.com/n1ckerr0r/shortener/internal/link/rediscache"
	"github.com/n1ckerr0r/shortener/internal/platform/clock"
	"github.com/n1ckerr0r/shortener/internal/platform/generator"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo, closeRepo := buildRepository(ctx)
	defer closeRepo()

	clk := clock.SystemClock{}
	gen := generator.RandomGenerator{}
	cache, cacheTTL, closeCache := buildResolveCache(ctx)
	defer closeCache()

	service := link.NewService(repo, gen, clk)
	if cache != nil {
		service = link.NewServiceWithCache(repo, gen, clk, cache, cacheTTL)
	}
	createHandler := httpapi.NewCreateLinkHandler(service)
	redirectHandler := httpapi.NewRedirectHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/links", createHandler)
	mux.Handle("/", redirectHandler)

	addr := getenv("SHORTENER_ADDR", ":8080")
	log.Printf("server started at %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func buildRepository(ctx context.Context) (link.Repository, func()) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("storage: memory")
		return memory.NewRepository(), func() {}
	}

	repo, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}

	log.Println("storage: postgres")
	return repo, func() {
		if err := repo.Close(); err != nil {
			log.Printf("close postgres: %v", err)
		}
	}
}

func buildResolveCache(ctx context.Context) (link.ResolveCache, time.Duration, func()) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Println("resolve cache: disabled")
		return nil, 0, func() {}
	}

	db, err := parseEnvInt("REDIS_DB", 0)
	if err != nil {
		log.Fatalf("parse REDIS_DB: %v", err)
	}
	cacheTTL, err := parseEnvDuration("REDIS_CACHE_TTL", 24*time.Hour)
	if err != nil {
		log.Fatalf("parse REDIS_CACHE_TTL: %v", err)
	}

	cache, err := rediscache.Open(ctx, rediscache.Options{
		Addr:     addr,
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
		Prefix:   os.Getenv("REDIS_KEY_PREFIX"),
	})
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	log.Printf("resolve cache: redis %s", addr)
	return cache, cacheTTL, func() {
		if err := cache.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func parseEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	return strconv.Atoi(value)
}

func parseEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	return time.ParseDuration(value)
}
