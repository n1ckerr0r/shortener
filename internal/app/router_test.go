package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/memory"
	"github.com/n1ckerr0r/shortener/internal/platform/clock"
	"github.com/n1ckerr0r/shortener/internal/platform/generator"
)

func TestRouterServesHealthEndpoints(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	service := link.NewService(memory.NewRepository(), generator.RandomGenerator{}, clock.SystemClock{})
	router := NewRouter(service, nil, logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if logs.Len() == 0 {
		t.Fatal("expected request log to be written")
	}
}
