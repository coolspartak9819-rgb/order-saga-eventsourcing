package infrastructure_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/infrastructure"
)

func TestMemoryEventStore_OptimisticConcurrencyAllowsOneWriter(t *testing.T) {
	const writers = 32

	store := infrastructure.NewMemoryEventStore()
	event := domain.OrderCreatedEvent{
		OrderID:    "order-concurrent",
		CustomerID: "customer-1",
		OccurredAt: time.Now().UTC(),
	}

	var waitGroup sync.WaitGroup
	results := make(chan error, writers)
	for index := 0; index < writers; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- store.SaveEvents(context.Background(), event.OrderID, []domain.DomainEvent{event}, 0)
		}()
	}
	waitGroup.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrConcurrencyConflict):
			conflicts++
		default:
			t.Fatalf("SaveEvents() unexpected error = %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("successful writes = %d, want 1", successes)
	}
	if conflicts != writers-1 {
		t.Fatalf("concurrency conflicts = %d, want %d", conflicts, writers-1)
	}

	events, err := store.LoadEvents(context.Background(), event.OrderID)
	if err != nil {
		t.Fatalf("LoadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("stored events = %d, want 1", len(events))
	}
}
