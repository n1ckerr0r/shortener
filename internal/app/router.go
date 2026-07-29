package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/httpapi"
)

type linkService interface {
	Create(ctx context.Context, req link.CreateRequest) (*link.CreateResponse, error)
	Resolve(ctx context.Context, req link.ResolveRequest) (*link.ResolveResponse, error)
}

func NewRouter(service linkService, readyChecks []ReadyCheck, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", LivenessHandler())
	mux.Handle("/readyz", ReadinessHandler(readyChecks, logger))
	mux.Handle("/metrics", MetricsHandler())
	mux.Handle("/links", httpapi.NewCreateLinkHandler(service))
	mux.Handle("/", httpapi.NewRedirectHandler(service))

	return trackInFlight(logRequests(mux, logger))
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func logRequests(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)
		recordHTTPRequest(r.Method, r.URL.Path, recorder.statusCode, time.Since(startedAt))

		logger.Info(
			"http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.statusCode),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)
	})
}
