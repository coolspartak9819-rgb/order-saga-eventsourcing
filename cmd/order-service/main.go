package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/api"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/clients"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/infrastructure"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/middleware"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/observability"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/outbox"
	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	eventStore := infrastructure.NewPostgresEventStore(db)
	paymentService, err := clients.NewPaymentClient(getEnv("PAYMENT_SERVICE_ADDR", "payment-service:50051"))
	if err != nil {
		log.Fatalf("create payment client: %v", err)
	}
	defer paymentService.Close()

	inventoryService, err := clients.NewInventoryClient(getEnv("INVENTORY_SERVICE_ADDR", "inventory-service:50052"))
	if err != nil {
		log.Fatalf("create inventory client: %v", err)
	}
	defer inventoryService.Close()

	saga := domain.NewOrderSagaOrchestrator(eventStore, paymentService, inventoryService)
	orderHandler := api.NewOrderHandler(eventStore, saga)
	outboxPublisher, err := outbox.NewPublisher(ctx, db, getEnv("NATS_URL", "nats://nats:4222"))
	if err != nil {
		log.Fatalf("create outbox publisher: %v", err)
	}
	defer outboxPublisher.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /api/orders", orderHandler.GetOrder)
	mux.HandleFunc("GET /api/orders/{id}", orderHandler.GetOrder)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeHealthResponse(w, http.StatusOK, "ok")
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		readyCtx, cancelReady := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancelReady()

		if err := db.PingContext(readyCtx); err != nil {
			writeHealthResponse(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		if outboxPublisher == nil || !outboxPublisher.Ready() {
			writeHealthResponse(w, http.StatusServiceUnavailable, "nats unavailable")
			return
		}

		writeHealthResponse(w, http.StatusOK, "ready")
	})
	metrics := observability.NewMetrics()
	mux.Handle("GET /metrics", metrics.Handler())

	idempotencyMiddleware := middleware.NewIdempotencyMiddleware(db)
	server := &http.Server{
		Addr:         ":8080",
		Handler:      metrics.Middleware(idempotencyMiddleware.Wrap(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		outboxPublisher.StartWorker(ctx)
	}()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("order service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		log.Fatalf("listen and serve: %v", err)
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
	<-workerDone

	log.Println("order service stopped")
}

func writeHealthResponse(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
