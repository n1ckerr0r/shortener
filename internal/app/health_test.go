package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	LivenessHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

func TestReadinessHandlerSuccess(t *testing.T) {
	handler := ReadinessHandler([]ReadyCheck{
		{
			Name: "dependency",
			Check: func(context.Context) error {
				return nil
			},
		},
	}, slog.Default())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadinessHandlerFailure(t *testing.T) {
	handler := ReadinessHandler([]ReadyCheck{
		{
			Name: "dependency",
			Check: func(context.Context) error {
				return errors.New("dependency is down")
			},
		},
	}, slog.Default())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
