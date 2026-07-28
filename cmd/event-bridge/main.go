package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	orderkafka "github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/kafka"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bridge, err := orderkafka.NewBridge(ctx, getEnv("NATS_URL", "nats://nats:4222"), getEnv("KAFKA_BROKERS", "kafka:9092"), getEnv("KAFKA_TOPIC", "orders.events"))
	if err != nil {
		log.Fatalf("create event bridge: %v", err)
	}
	defer bridge.Close()

	log.Println("event bridge started")
	if err := bridge.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("event bridge stopped: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
