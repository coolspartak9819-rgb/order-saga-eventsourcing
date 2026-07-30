package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/coolspartak9819-rgb/queuelens/internal/events"
	"github.com/coolspartak9819-rgb/queuelens/internal/store"
	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	jobStore, err := store.New(ctx, env("DATABASE_URL", "postgres://queuelens:queuelens@localhost:5432/queuelens"))
	if err != nil {
		log.Fatal(err)
	}
	defer jobStore.Close()
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","), Topic: env("KAFKA_TOPIC", "job-events"), GroupID: env("KAFKA_GROUP", "queuelens-audit"), MinBytes: 1, MaxBytes: 10e6})
	defer reader.Close()
	log.Println("QueueLens audit consumer started")
	for {
		message, err := reader.FetchMessage(ctx)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			log.Printf("fetch Kafka event: %v", err)
			continue
		}
		var event events.JobEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			log.Printf("decode Kafka event: %v", err)
			continue
		}
		if err := jobStore.RecordEvent(ctx, event, message.Partition, message.Offset); err != nil {
			log.Printf("store audit event: %v", err)
			continue
		}
		if err := reader.CommitMessages(ctx, message); err != nil {
			log.Printf("commit Kafka offset: %v", err)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
