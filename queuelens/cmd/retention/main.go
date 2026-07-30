package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	storeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	jobStore, err := store.New(storeCtx, env("DATABASE_URL", "postgres://queuelens:queuelens@localhost:5432/queuelens"))
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer jobStore.Close()

	retentionDays, err := strconv.Atoi(env("RETENTION_DAYS", "30"))
	if err != nil || retentionDays < 1 {
		log.Fatal("RETENTION_DAYS must be a positive integer")
	}

	log.Printf("QueueLens retention worker started: retention_days=%d", retentionDays)
	purge := func() {
		purgeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		count, err := jobStore.PurgeOld(purgeCtx, retentionDays)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			log.Printf("purge old jobs: %v", err)
			return
		}
		if count > 0 {
			log.Printf("purged jobs: count=%d", count)
		}
	}

	purge()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
