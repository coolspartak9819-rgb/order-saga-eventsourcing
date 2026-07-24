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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /api/orders", orderHandler.GetOrder)
	mux.HandleFunc("GET /api/orders/{id}", orderHandler.GetOrder)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("order service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}

	log.Println("order service stopped")
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
