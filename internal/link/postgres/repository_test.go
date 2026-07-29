package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
)

func TestRepositorySaveFindExists(t *testing.T) {
	dsn := os.Getenv("SHORTENER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SHORTENER_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			t.Logf("close db: %v", err)
		}
	}()

	repo := NewRepository(db)
	if err = repo.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err = repo.Init(ctx); err != nil {
		t.Fatal(err)
	}

	codeValue := fmt.Sprintf("test%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM links WHERE short_code = $1", codeValue)
	})

	code, err := link.NewShortCode(codeValue)
	if err != nil {
		t.Fatal(err)
	}
	originalURL, err := link.NewOriginalURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	shortLink, err := link.NewShortLink(
		code,
		originalURL,
		time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		link.NewExpiration(&expiresAt),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = repo.Save(ctx, shortLink); err != nil {
		t.Fatalf("save: %v", err)
	}

	exists, err := repo.Exists(ctx, code)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Fatal("expected link to exist")
	}

	found, err := repo.Find(ctx, code)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.OriginalURL().Value() != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", found.OriginalURL().Value())
	}
}
