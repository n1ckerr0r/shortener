package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/memory"
)

type fixedGenerator struct {
	code string
}

func (g fixedGenerator) Generate() (link.ShortCode, error) {
	return link.NewShortCode(g.code)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestCreateLinkHandlerSuccess(t *testing.T) {
	service := newTestService(t)
	handler := NewCreateLinkHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"url":"https://example.com"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp CreateLinkResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ShortCode != "abc123" {
		t.Fatalf("expected abc123, got %s", resp.ShortCode)
	}
}

func TestCreateLinkHandlerInvalidJSON(t *testing.T) {
	handler := NewCreateLinkHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`invalid-json`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLinkHandlerRejectsMultipleJSONDocuments(t *testing.T) {
	handler := NewCreateLinkHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"url":"https://example.com"}{"url":"https://evil.example.com"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLinkHandlerInvalidURL(t *testing.T) {
	handler := NewCreateLinkHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"url":"not a url"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLinkHandlerMethodNotAllowed(t *testing.T) {
	handler := NewCreateLinkHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodGet, "/links", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("expected Allow POST, got %q", rec.Header().Get("Allow"))
	}
}

func TestRedirectHandlerSuccess(t *testing.T) {
	service := newTestService(t)
	_, err := service.Create(context.Background(), link.CreateRequest{
		OriginalURL: "https://example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRedirectHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "https://example.com" {
		t.Fatalf("expected redirect to https://example.com, got %s", location)
	}
}

func TestRedirectHandlerNotFound(t *testing.T) {
	handler := NewRedirectHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRedirectHandlerRootNotFound(t *testing.T) {
	handler := NewRedirectHandler(newTestService(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func newTestService(t *testing.T) *link.Service {
	t.Helper()

	return link.NewService(
		memory.NewRepository(),
		fixedGenerator{code: "abc123"},
		fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
	)
}
