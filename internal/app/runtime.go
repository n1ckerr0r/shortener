package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/n1ckerr0r/shortener/internal/link"
	"github.com/n1ckerr0r/shortener/internal/link/grpcapi"
	"github.com/n1ckerr0r/shortener/internal/platform/clock"
	"github.com/n1ckerr0r/shortener/internal/platform/generator"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return RunWithConfig(ctx, cfg, logger)
}

func RunWithConfig(ctx context.Context, cfg Config, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	startupCtx, cancelStartup := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancelStartup()

	repoHandle, err := buildRepository(startupCtx, cfg, logger)
	if err != nil {
		return err
	}
	defer repoHandle.close(logger)

	cacheHandle, err := buildResolveCache(startupCtx, cfg, logger)
	if err != nil {
		return err
	}
	defer cacheHandle.close(logger)

	publisherHandle, err := buildPublishers(cfg, logger)
	if err != nil {
		return err
	}
	defer publisherHandle.close(logger)

	clk := clock.SystemClock{}
	codeGenerator := generator.RandomGenerator{}
	options := []link.ServiceOption{
		link.WithClickPublisher(publisherHandle.clickPublisher),
		link.WithURLCheckPublisher(publisherHandle.urlCheckPublisher),
	}
	if cacheHandle.cache != nil {
		options = append(options, link.WithResolveCache(cacheHandle.cache, cacheHandle.ttl))
	}
	linkService := link.NewServiceWithOptions(repoHandle.repository, codeGenerator, clk, options...)
	grpcServer := grpc.NewServer()
	grpcapi.Register(grpcServer, linkService)

	readyChecks := append(repoHandle.readyChecks, cacheHandle.readyChecks...)
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      NewRouter(linkService, readyChecks, logger),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	if err = serve(ctx, server, grpcServer, cfg.GRPC.Addr, cfg.ShutdownTimeout, logger); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func serve(
	ctx context.Context,
	httpServer *http.Server,
	grpcServer *grpc.Server,
	grpcAddr string,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http_server_started", slog.String("addr", httpServer.Addr))
		errCh <- httpServer.ListenAndServe()
	}()

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	go func() {
		logger.Info("grpc_server_started", slog.String("addr", grpcAddr))
		errCh <- grpcServer.Serve(grpcListener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("servers_stopping", slog.String("reason", ctx.Err().Error()))
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	grpcServer.GracefulStop()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, grpc.ErrServerStopped) {
			continue
		}

		return err
	}

	logger.Info("servers_stopped")
	return nil
}
