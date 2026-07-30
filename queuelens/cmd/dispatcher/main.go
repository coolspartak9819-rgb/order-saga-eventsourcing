package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/queue"
	"github.com/coolspartak9819-rgb/queuelens/internal/store"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	jobStore, err := store.New(ctx, env("DATABASE_URL", "postgres://queuelens:queuelens@localhost:5432/queuelens"))
	if err != nil {
		log.Fatal(err)
	}
	defer jobStore.Close()
	jobQueue := queue.New(env("REDIS_ADDR", "localhost:6379"), env("QUEUE_STREAM", "jobs"), "dispatcher", "dispatcher")
	defer jobQueue.Close()
	if err := jobQueue.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("QueueLens outbox dispatcher started")
	for {
		message, err := jobStore.ClaimOutbox(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			log.Printf("claim outbox: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := jobQueue.Enqueue(ctx, message.Job); err != nil {
			log.Printf("publish outbox %d: %v", message.ID, err)
			if retryErr := jobStore.MarkOutboxRetry(ctx, message.ID, err.Error(), retryDelay(message.Attempts)); retryErr != nil {
				log.Printf("mark outbox retry: %v", retryErr)
			}
			continue
		}
		if err := jobStore.MarkOutboxPublished(ctx, message.ID); err != nil {
			log.Printf("mark outbox published: %v", err)
		}
	}
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<attempts) * time.Second
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
