package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/httpapi"
	"github.com/n1ckerr0r/shortener/internal/link/memory"
	"github.com/n1ckerr0r/shortener/internal/link/postgres"
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

	service := link.NewService(repo, gen, clk)
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

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
