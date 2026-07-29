package rediscache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
)

func TestCacheSetGet(t *testing.T) {
	addr := os.Getenv("SHORTENER_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("SHORTENER_TEST_REDIS_ADDR is not set")
	}

	ctx := context.Background()
	cache, err := Open(ctx, Options{
		Addr:   addr,
		Prefix: "shortener:test:",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = cache.Close(); err != nil {
			t.Logf("close cache: %v", err)
		}
	}()

	code, err := link.NewShortCode(fmt.Sprintf("test%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	originalURL, err := link.NewOriginalURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if err = cache.Set(ctx, code, originalURL, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	got, ok, err := cache.Get(ctx, code)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected cached value")
	}
	if got.Value() != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", got.Value())
	}
}
