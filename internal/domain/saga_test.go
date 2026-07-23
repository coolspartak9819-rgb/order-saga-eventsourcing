package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/infrastructure"
)

func TestSaga_SuccessPath(t *testing.T) {
	t.Run("writes all events and completes order", func(t *testing.T) {
		ctx := context.Background()
		store := infrastructure.NewMemoryEventStore()
		paymentService := &MockPaymentService{}
		inventoryService := &MockInventoryService{}
		saga := domain.NewOrderSagaOrchestrator(store, paymentService, inventoryService)
		order := newTestOrder(t)

		if err := saga.ExecuteOrderSaga(ctx, order); err != nil {
			t.Fatalf("ExecuteOrderSaga() error = %v", err)
		}

		assertEventTypes(t, store, order.ID, []string{
			domain.EventTypeOrderCreated,
			domain.EventTypePaymentAuthorized,
			domain.EventTypeInventoryReserved,
			domain.EventTypeOrderCompleted,
		})

		if order.Status != domain.OrderStatusCompleted {
			t.Fatalf("order status = %q, want %q", order.Status, domain.OrderStatusCompleted)
		}
		if paymentService.ProcessPaymentCalls != 1 {
			t.Fatalf("ProcessPayment calls = %d, want 1", paymentService.ProcessPaymentCalls)
		}
		if paymentService.RefundPaymentCalls != 0 {
			t.Fatalf("RefundPayment calls = %d, want 0", paymentService.RefundPaymentCalls)
		}
		if inventoryService.ReserveInventoryCalls != 1 {
			t.Fatalf("ReserveInventory calls = %d, want 1", inventoryService.ReserveInventoryCalls)
		}
	})
}

func TestSaga_PaymentFailed(t *testing.T) {
	t.Run("writes failure events and skips inventory", func(t *testing.T) {
		ctx := context.Background()
		store := infrastructure.NewMemoryEventStore()
		paymentService := &MockPaymentService{
			ProcessPaymentErr: errors.New("payment declined"),
		}
		inventoryService := &MockInventoryService{}
		saga := domain.NewOrderSagaOrchestrator(store, paymentService, inventoryService)
		order := newTestOrder(t)

		if err := saga.ExecuteOrderSaga(ctx, order); err == nil {
			t.Fatal("ExecuteOrderSaga() error = nil, want payment error")
		}

		assertEventTypes(t, store, order.ID, []string{
			domain.EventTypeOrderCreated,
			domain.EventTypePaymentFailed,
			domain.EventTypeOrderCancelled,
		})

		if order.Status != domain.OrderStatusCancelled {
			t.Fatalf("order status = %q, want %q", order.Status, domain.OrderStatusCancelled)
		}
		if paymentService.ProcessPaymentCalls != 1 {
			t.Fatalf("ProcessPayment calls = %d, want 1", paymentService.ProcessPaymentCalls)
		}
		if paymentService.RefundPaymentCalls != 0 {
			t.Fatalf("RefundPayment calls = %d, want 0", paymentService.RefundPaymentCalls)
		}
		if inventoryService.ReserveInventoryCalls != 0 {
			t.Fatalf("ReserveInventory calls = %d, want 0", inventoryService.ReserveInventoryCalls)
		}
	})
}

func TestSaga_InventoryFailed_CompensatingTransaction(t *testing.T) {
	t.Run("writes inventory failure, refunds payment, and cancels order", func(t *testing.T) {
		ctx := context.Background()
		store := infrastructure.NewMemoryEventStore()
		paymentService := &MockPaymentService{}
		inventoryService := &MockInventoryService{
			ReserveInventoryErr: errors.New("not enough inventory"),
		}
		saga := domain.NewOrderSagaOrchestrator(store, paymentService, inventoryService)
		order := newTestOrder(t)

		if err := saga.ExecuteOrderSaga(ctx, order); err == nil {
			t.Fatal("ExecuteOrderSaga() error = nil, want inventory error")
		}

		assertEventTypes(t, store, order.ID, []string{
			domain.EventTypeOrderCreated,
			domain.EventTypePaymentAuthorized,
			domain.EventTypeInventoryFailed,
			domain.EventTypeOrderCancelled,
		})

		if order.Status != domain.OrderStatusCancelled {
			t.Fatalf("order status = %q, want %q", order.Status, domain.OrderStatusCancelled)
		}
		if paymentService.ProcessPaymentCalls != 1 {
			t.Fatalf("ProcessPayment calls = %d, want 1", paymentService.ProcessPaymentCalls)
		}
		if paymentService.RefundPaymentCalls != 1 {
			t.Fatalf("RefundPayment calls = %d, want 1", paymentService.RefundPaymentCalls)
		}
		if inventoryService.ReserveInventoryCalls != 1 {
			t.Fatalf("ReserveInventory calls = %d, want 1", inventoryService.ReserveInventoryCalls)
		}
	})
}

type MockPaymentService struct {
	ProcessPaymentErr   error
	RefundPaymentErr    error
	ProcessPaymentCalls int
	RefundPaymentCalls  int
}

func (m *MockPaymentService) ProcessPayment(ctx context.Context, orderID string, amount float64) error {
	m.ProcessPaymentCalls++
	return m.ProcessPaymentErr
}

func (m *MockPaymentService) RefundPayment(ctx context.Context, orderID string, amount float64) error {
	m.RefundPaymentCalls++
	return m.RefundPaymentErr
}

type MockInventoryService struct {
	ReserveInventoryErr   error
	ReleaseInventoryErr   error
	ReserveInventoryCalls int
	ReleaseInventoryCalls int
}

func (m *MockInventoryService) ReserveInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	m.ReserveInventoryCalls++
	return m.ReserveInventoryErr
}

func (m *MockInventoryService) ReleaseInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	m.ReleaseInventoryCalls++
	return m.ReleaseInventoryErr
}

func newTestOrder(t *testing.T) *domain.Order {
	t.Helper()

	order, err := domain.NewOrder("order-1", "customer-1", []domain.OrderItem{
		{ProductID: "product-1", Quantity: 2, Price: 100},
		{ProductID: "product-2", Quantity: 1, Price: 50},
	})
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}

	return order
}

func assertEventTypes(t *testing.T, store domain.EventStore, orderID string, want []string) {
	t.Helper()

	events, err := store.LoadEvents(context.Background(), orderID)
	if err != nil {
		t.Fatalf("LoadEvents() error = %v", err)
	}
	if len(events) != len(want) {
		t.Fatalf("events count = %d, want %d", len(events), len(want))
	}

	for i, event := range events {
		if event.EventType() != want[i] {
			t.Fatalf("events[%d] type = %q, want %q", i, event.EventType(), want[i])
		}
	}
}
