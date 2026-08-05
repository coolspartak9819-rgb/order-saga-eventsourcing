package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/httpapi"
	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/service"
	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	queue := store.NewMemoryQueue()
	deliveries := service.NewDeliveryService(queue, http.DefaultClient, logger)
	server := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           httpapi.NewRouter(deliveries),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go deliveries.Run(ctx, 8)

	go func() {
		logger.Info("webhook delivery api started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
