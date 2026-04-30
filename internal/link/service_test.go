package link

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	links     map[string]*ShortLink
	saveCalls int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		links: make(map[string]*ShortLink),
	}
}

func (r *fakeRepository) Save(_ context.Context, shortLink *ShortLink) error {
	r.saveCalls++
	r.links[shortLink.ShortCode().Value()] = shortLink
	return nil
}

func (r *fakeRepository) Find(_ context.Context, code ShortCode) (*ShortLink, error) {
	shortLink, ok := r.links[code.Value()]
	if !ok {
		return nil, ErrNotFound
	}

	return shortLink, nil
}

func (r *fakeRepository) Exists(_ context.Context, code ShortCode) (bool, error) {
	_, ok := r.links[code.Value()]
	return ok, nil
}

type fakeGenerator struct {
	codes []string
	index int
}

func (g *fakeGenerator) Generate() (ShortCode, error) {
	if g.index >= len(g.codes) {
		return ShortCode{}, ErrShortCodeCollision
	}

	code, err := NewShortCode(g.codes[g.index])
	g.index++
	return code, err
}

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time {
	return c.now
}

func TestServiceCreateSuccess(t *testing.T) {
	repo := newFakeRepository()
	clock := fakeClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	service := NewService(repo, &fakeGenerator{codes: []string{"abc123"}}, clock)

	resp, err := service.Create(context.Background(), CreateRequest{
		OriginalURL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ShortCode != "abc123" {
		t.Fatalf("expected abc123, got %s", resp.ShortCode)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save call, got %d", repo.saveCalls)
	}
	if repo.links["abc123"].CreatedAt() != clock.now {
		t.Fatalf("created time was not taken from clock")
	}
}

func TestServiceCreateInvalidURLDoesNotSave(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, &fakeGenerator{codes: []string{"abc123"}}, fakeClock{})

	_, err := service.Create(context.Background(), CreateRequest{
		OriginalURL: "not a url",
	})
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("expected ErrInvalidURL, got %v", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save calls, got %d", repo.saveCalls)
	}
}

func TestServiceCreateRejectsExpirationBeforeCreation(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
	repo := newFakeRepository()
	service := NewService(repo, &fakeGenerator{codes: []string{"abc123"}}, fakeClock{now: now})

	_, err := service.Create(context.Background(), CreateRequest{
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
	})
	if !errors.Is(err, ErrInvalidExpirationDate) {
		t.Fatalf("expected ErrInvalidExpirationDate, got %v", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save calls, got %d", repo.saveCalls)
	}
}

func TestServiceCreateRetriesShortCodeCollision(t *testing.T) {
	repo := newFakeRepository()
	occupiedCode, err := NewShortCode("abc123")
	if err != nil {
		t.Fatal(err)
	}
	originalURL, err := NewOriginalURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	occupied, err := NewShortLink(occupiedCode, originalURL, time.Now(), NewExpiration(nil))
	if err != nil {
		t.Fatal(err)
	}
	repo.links["abc123"] = occupied

	service := NewService(repo, &fakeGenerator{codes: []string{"abc123", "def456"}}, fakeClock{now: time.Now()})
	resp, err := service.Create(context.Background(), CreateRequest{
		OriginalURL: "https://example.org",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ShortCode != "def456" {
		t.Fatalf("expected def456, got %s", resp.ShortCode)
	}
}

func TestServiceResolveSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	shortLink := mustShortLink(t, "abc123", "https://example.com", now, nil)
	if err := repo.Save(context.Background(), shortLink); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, &fakeGenerator{}, fakeClock{now: now})
	resp, err := service.Resolve(context.Background(), ResolveRequest{Code: "abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OriginalURL != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", resp.OriginalURL)
	}
}

func TestServiceResolveExpired(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	expiresAt := createdAt.Add(time.Hour)
	repo := newFakeRepository()
	shortLink := mustShortLink(t, "abc123", "https://example.com", createdAt, &expiresAt)
	if err := repo.Save(context.Background(), shortLink); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, &fakeGenerator{}, fakeClock{now: expiresAt.Add(time.Second)})
	_, err := service.Resolve(context.Background(), ResolveRequest{Code: "abc123"})
	if !errors.Is(err, ErrExpiredLink) {
		t.Fatalf("expected ErrExpiredLink, got %v", err)
	}
}

func TestServiceResolveBlocked(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	shortLink := mustShortLink(t, "abc123", "https://example.com", now, nil)
	shortLink.Block()
	if err := repo.Save(context.Background(), shortLink); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, &fakeGenerator{}, fakeClock{now: now})
	_, err := service.Resolve(context.Background(), ResolveRequest{Code: "abc123"})
	if !errors.Is(err, ErrBlockedLink) {
		t.Fatalf("expected ErrBlockedLink, got %v", err)
	}
}

func mustShortLink(t *testing.T, codeValue, urlValue string, createdAt time.Time, expiresAt *time.Time) *ShortLink {
	t.Helper()

	code, err := NewShortCode(codeValue)
	if err != nil {
		t.Fatal(err)
	}
	originalURL, err := NewOriginalURL(urlValue)
	if err != nil {
		t.Fatal(err)
	}
	shortLink, err := NewShortLink(code, originalURL, createdAt, NewExpiration(expiresAt))
	if err != nil {
		t.Fatal(err)
	}

	return shortLink
}
