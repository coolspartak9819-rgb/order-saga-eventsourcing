package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrOrderRequired            = errors.New("order is required")
	ErrEventStoreRequired       = errors.New("event store is required")
	ErrPaymentServiceRequired   = errors.New("payment service is required")
	ErrInventoryServiceRequired = errors.New("inventory service is required")
)

type PaymentService interface {
	ProcessPayment(ctx context.Context, orderID string, amount float64) error
	RefundPayment(ctx context.Context, orderID string, amount float64) error
}

type InventoryService interface {
	ReserveInventory(ctx context.Context, orderID string, items []OrderItem) error
	ReleaseInventory(ctx context.Context, orderID string, items []OrderItem) error
}

type OrderSagaOrchestrator struct {
	EventStore       EventStore
	PaymentService   PaymentService
	InventoryService InventoryService
}

func NewOrderSagaOrchestrator(
	eventStore EventStore,
	paymentService PaymentService,
	inventoryService InventoryService,
) *OrderSagaOrchestrator {
	return &OrderSagaOrchestrator{
		EventStore:       eventStore,
		PaymentService:   paymentService,
		InventoryService: inventoryService,
	}
}

func (o *OrderSagaOrchestrator) ExecuteOrderSaga(ctx context.Context, order *Order) error {
	if err := o.validate(); err != nil {
		return err
	}
	if order == nil {
		return ErrOrderRequired
	}

	if err := o.saveUncommittedEvents(ctx, order); err != nil {
		return err
	}

	if err := o.PaymentService.ProcessPayment(ctx, order.ID, order.TotalAmount); err != nil {
		order.RaiseEvent(PaymentFailedEvent{
			OrderID:    order.ID,
			Reason:     err.Error(),
			OccurredAt: time.Now().UTC(),
		})
		order.RaiseEvent(OrderCancelledEvent{
			OrderID:    order.ID,
			Reason:     "payment failed",
			OccurredAt: time.Now().UTC(),
		})

		if saveErr := o.saveUncommittedEvents(ctx, order); saveErr != nil {
			return fmt.Errorf("save payment failure events: %w", saveErr)
		}

		return fmt.Errorf("process payment: %w", err)
	}

	order.RaiseEvent(PaymentAuthorizedEvent{
		OrderID:    order.ID,
		Amount:     int64(order.TotalAmount),
		OccurredAt: time.Now().UTC(),
	})

	if err := o.saveUncommittedEvents(ctx, order); err != nil {
		return err
	}

	if err := o.InventoryService.ReserveInventory(ctx, order.ID, order.Items); err != nil {
		order.RaiseEvent(InventoryFailedEvent{
			OrderID:    order.ID,
			Items:      cloneOrderItems(order.Items),
			Reason:     err.Error(),
			OccurredAt: time.Now().UTC(),
		})

		refundErr := o.PaymentService.RefundPayment(ctx, order.ID, order.TotalAmount)

		order.RaiseEvent(OrderCancelledEvent{
			OrderID:    order.ID,
			Reason:     "inventory reservation failed",
			OccurredAt: time.Now().UTC(),
		})

		if saveErr := o.saveUncommittedEvents(ctx, order); saveErr != nil {
			return fmt.Errorf("save inventory failure events: %w", saveErr)
		}

		if refundErr != nil {
			return fmt.Errorf("reserve inventory: %w; refund payment: %v", err, refundErr)
		}

		return fmt.Errorf("reserve inventory: %w", err)
	}

	order.RaiseEvent(InventoryReservedEvent{
		OrderID:    order.ID,
		Items:      cloneOrderItems(order.Items),
		OccurredAt: time.Now().UTC(),
	})
	order.RaiseEvent(OrderCompletedEvent{
		OrderID:    order.ID,
		OccurredAt: time.Now().UTC(),
	})

	return o.saveUncommittedEvents(ctx, order)
}

func (o *OrderSagaOrchestrator) validate() error {
	if o.EventStore == nil {
		return ErrEventStoreRequired
	}
	if o.PaymentService == nil {
		return ErrPaymentServiceRequired
	}
	if o.InventoryService == nil {
		return ErrInventoryServiceRequired
	}
	return nil
}

func (o *OrderSagaOrchestrator) saveUncommittedEvents(ctx context.Context, order *Order) error {
	if len(order.UncommittedEvents) == 0 {
		return nil
	}

	expectedVersion := order.Version - len(order.UncommittedEvents)
	if err := o.EventStore.SaveEvents(ctx, order.ID, order.UncommittedEvents, expectedVersion); err != nil {
		return err
	}

	order.ClearUncommittedEvents()
	return nil
}
