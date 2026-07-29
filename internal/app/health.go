package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

type ReadyCheck struct {
	Name  string
	Check func(context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

func LivenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeHealthJSON(w, http.StatusOK, "ok")
	})
}

func ReadinessHandler(checks []ReadyCheck, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		for _, check := range checks {
			if err := check.Check(ctx); err != nil {
				logger.Error("readiness_check_failed", slog.String("check", check.Name), slog.Any("error", err))
				writeHealthJSON(w, http.StatusServiceUnavailable, "unavailable")
				return
			}
		}

		writeHealthJSON(w, http.StatusOK, "ok")
	})
}

func writeHealthJSON(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: status})
}
