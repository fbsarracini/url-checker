package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/fbsarracini/url-checker/internal/checker"
	"github.com/fbsarracini/url-checker/internal/config"
	"github.com/fbsarracini/url-checker/internal/handler"
	"github.com/fbsarracini/url-checker/internal/observability"
)

type metricsAdapter struct {
	m *observability.Metrics
}

func (a *metricsAdapter) ObserveCheck(result string) {
	a.m.URLChecksTotal.WithLabelValues(result).Inc()
}

var _ handler.CheckMetrics = (*metricsAdapter)(nil)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	reg := prometheus.NewRegistry()
	metrics := observability.NewMetrics(reg)

	httpChecker := checker.New(cfg.CheckMaxTimeout, cfg.CheckUserAgent)

	var isShuttingDown atomic.Bool
	healthH := handler.NewHealthHandler(&isShuttingDown)
	checkH := handler.NewCheckHandler(
		httpChecker,
		logger,
		&metricsAdapter{m: metrics},
		handler.CheckConfig{
			MaxURLs:        cfg.MaxURLsPerRequest,
			MaxChecks:      cfg.MaxChecksPerRequest,
			DefaultTimeout: cfg.CheckDefaultTimeout,
			MaxTimeout:     cfg.CheckMaxTimeout,
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthH.Liveness)
	mux.HandleFunc("/readyz", healthH.Readiness)
	mux.Handle("/check", chain(
		checkH,
		handler.RequestID,
		handler.Logging(logger),
		handler.Recover(logger),
		handler.ConcurrencyLimit(cfg.MaxConcurrentRequests),
	))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := observability.StartMetricsServer(ctx, cfg.MetricsPort, cfg.ShutdownTimeout, reg); err != nil {
			logger.Error("metrics server error", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("server starting", "port", cfg.Port, "metrics_port", cfg.MetricsPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, draining")
	isShuttingDown.Store(true)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	wg.Wait()
	logger.Info("server stopped")
}

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
