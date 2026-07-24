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
	paymentService := FakePaymentService{}
	inventoryService := FakeInventoryService{}
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

type FakePaymentService struct{}

func (FakePaymentService) ProcessPayment(ctx context.Context, orderID string, amount float64) error {
	return ctx.Err()
}

func (FakePaymentService) RefundPayment(ctx context.Context, orderID string, amount float64) error {
	return ctx.Err()
}

type FakeInventoryService struct{}

func (FakeInventoryService) ReserveInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	return ctx.Err()
}

func (FakeInventoryService) ReleaseInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	return ctx.Err()
}
