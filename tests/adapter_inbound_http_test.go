package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	inhttp "github.com/n1ckerr0r/shortener/adapters/inbound/http"
	"github.com/n1ckerr0r/shortener/core/application/create_link"
	resolve_link2 "github.com/n1ckerr0r/shortener/core/application/resolve_link"
	"github.com/n1ckerr0r/shortener/core/domain/link"
	"github.com/n1ckerr0r/shortener/infrastructure/repository"
)

type fakeClock struct {
	now time.Time
}

func (fc fakeClock) Now() time.Time {
	return fc.now
}

type fakeGenerator struct {
	code string
}

func (gen fakeGenerator) Generate() (link.ShortCode, error) {
	shortCode, err := link.NewShortCode("abc123")
	if err != nil {
		return link.ShortCode{}, err
	}
	return shortCode, nil
}

// Тесты для CreateLinkHandler
func TestCreateLinkHandler_ServeHTTP_Success(t *testing.T) {
	repo := repository.NewMemoryRepository()
	clock := fakeClock{now: time.Now()}
	gen := fakeGenerator{code: "abc123"}

	service := create_link.NewService(repo, gen, clock)
	handler := inhttp.NewCreateLinkHandler(service)

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp inhttp.CreateLinkResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ShortCode != "abc123" {
		t.Fatalf("expected short code abc123, got %s", resp.ShortCode)
	}
}

func TestCreateLinkHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	repo := repository.NewMemoryRepository()
	clock := fakeClock{now: time.Now()}
	gen := fakeGenerator{code: "abc123"}

	service := create_link.NewService(repo, gen, clock)
	handler := inhttp.NewCreateLinkHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`invalid-json`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// Тесты для RedirectHandler
//func TestRedirectHandler_ServeHTTP_Success(t *testing.T) {
//	repo := repository.NewMemoryRepository()
//	now := time.Now()
//
//	var shortCode link.ShortCode
//	var originalURL link.OriginalURL
//	var expiration link.Expiration
//
//	shortCode, err := link.NewShortCode("abc123")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	originalURL, err = link.NewOriginalURL("https://example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	nowTime := time.Now()
//	expiration = link.NewExpiration(&nowTime)
//
//	linkObj, _ := link.NewShortLink(
//		shortCode,
//		originalURL,
//		time.Now(),
//		expiration,
//	)
//
//	if err = repo.Save(linkObj); err != nil {
//		log.Fatal(err)
//	}
//
//	clock := fakeClock{now: now}
//	service := resolve_link2.NewService(repo, clock)
//	handler := inhttp.NewRedirectHandler(service)
//
//	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
//	rec := httptest.NewRecorder()
//
//	handler.ServeHTTP(rec, req)
//
//	if rec.Code != http.StatusFound {
//		t.Fatalf("expected 302, got %d", rec.Code)
//	}
//
//	location := rec.Header().Get("Location")
//	if location != "https://example.com" {
//		t.Fatalf("expected redirect to example.com, got %s", location)
//	}
//}

func TestRedirectHandler_ServeHTTP_NotFound(t *testing.T) {
	repo := repository.NewMemoryRepository()
	clock := fakeClock{now: time.Now()}

	service := resolve_link2.NewService(repo, clock)
	handler := inhttp.NewRedirectHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
