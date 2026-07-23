package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/infrastructure"
)

func TestOrderEventSourcingFullCycle(t *testing.T) {
	ctx := context.Background()
	store := infrastructure.NewMemoryEventStore()

	items := []domain.OrderItem{
		{ProductID: "product-1", Quantity: 2, Price: 100},
		{ProductID: "product-2", Quantity: 1, Price: 50},
	}

	order, err := domain.NewOrder("order-1", "customer-1", items)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}

	now := time.Now().UTC()
	order.RaiseEvent(domain.PaymentAuthorizedEvent{
		OrderID:    order.ID,
		PaymentID:  "payment-1",
		Amount:     250,
		Currency:   "USD",
		OccurredAt: now,
	})
	order.RaiseEvent(domain.InventoryReservedEvent{
		OrderID:       order.ID,
		ReservationID: "reservation-1",
		Items:         items,
		OccurredAt:    now,
	})

	if err := store.SaveEvents(ctx, order.ID, order.UncommittedEvents, 0); err != nil {
		t.Fatalf("SaveEvents() error = %v", err)
	}

	loadedEvents, err := store.LoadEvents(ctx, order.ID)
	if err != nil {
		t.Fatalf("LoadEvents() error = %v", err)
	}

	restoredOrder := &domain.Order{}
	restoredOrder.LoadFromHistory(loadedEvents)

	if restoredOrder.Status != domain.OrderStatusReserved {
		t.Fatalf("restored status = %q, want %q", restoredOrder.Status, domain.OrderStatusReserved)
	}

	if restoredOrder.Version != order.Version {
		t.Fatalf("restored version = %d, want %d", restoredOrder.Version, order.Version)
	}

	if len(restoredOrder.UncommittedEvents) != 0 {
		t.Fatalf("restored uncommitted events = %d, want 0", len(restoredOrder.UncommittedEvents))
	}
}
