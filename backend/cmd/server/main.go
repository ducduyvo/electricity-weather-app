package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	weatherhttp "weather-retailer/backend/internal/transport/http"
	"weather-retailer/backend/internal/weather"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	client := &http.Client{Timeout: cfg.ClientTimeout}
	defer client.CloseIdleConnections()
	upstream := weather.OpenMeteo{Client: client, BaseURL: cfg.OpenMeteoURL, WindInsights: &cfg.WindInsights}
	provider := &weather.Cached{Next: upstream, TTL: cfg.CacheTTL}
	handler := weatherhttp.NewHandler(provider)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", weatherhttp.Health)
	mux.HandleFunc("GET /api/weather", handler.Weather)
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           cors(cfg.AllowedOrigin, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.ClientTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	slog.Info("API listening", "address", cfg.Address)

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		// A second signal may terminate immediately while graceful shutdown runs.
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
