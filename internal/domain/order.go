package domain

import (
	"errors"
	"time"
)

const (
	OrderStatusCreated   = "CREATED"
	OrderStatusPaid      = "PAID"
	OrderStatusReserved  = "RESERVED"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusCompleted = "COMPLETED"
)

var (
	ErrOrderIDRequired    = errors.New("order id is required")
	ErrCustomerIDRequired = errors.New("customer id is required")
	ErrOrderItemsRequired = errors.New("order items are required")
)

type Order struct {
	ID                string
	CustomerID        string
	Status            string
	Items             []OrderItem
	TotalAmount       float64
	Version           int
	UncommittedEvents []DomainEvent
}

func NewOrder(id, customerID string, items []OrderItem) (*Order, error) {
	if id == "" {
		return nil, ErrOrderIDRequired
	}
	if customerID == "" {
		return nil, ErrCustomerIDRequired
	}
	if len(items) == 0 {
		return nil, ErrOrderItemsRequired
	}

	order := &Order{}
	order.RaiseEvent(OrderCreatedEvent{
		OrderID:    id,
		CustomerID: customerID,
		Items:      cloneOrderItems(items),
		Amount:     calculateTotalAmount(items),
		OccurredAt: time.Now().UTC(),
	})

	return order, nil
}

func (o *Order) Apply(event DomainEvent) {
	applied := true

	switch e := event.(type) {
	case OrderCreatedEvent:
		o.applyOrderCreated(e)
	case *OrderCreatedEvent:
		if e == nil {
			applied = false
			break
		}
		o.applyOrderCreated(*e)
	case PaymentAuthorizedEvent:
		o.Status = OrderStatusPaid
	case *PaymentAuthorizedEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusPaid
	case InventoryReservedEvent:
		o.Status = OrderStatusReserved
	case *InventoryReservedEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusReserved
	case PaymentFailedEvent:
		o.Status = OrderStatusCancelled
	case *PaymentFailedEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusCancelled
	case InventoryFailedEvent:
		o.Status = OrderStatusCancelled
	case *InventoryFailedEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusCancelled
	case OrderCompletedEvent:
		o.Status = OrderStatusCompleted
	case *OrderCompletedEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusCompleted
	case OrderCancelledEvent:
		o.Status = OrderStatusCancelled
	case *OrderCancelledEvent:
		if e == nil {
			applied = false
			break
		}
		o.Status = OrderStatusCancelled
	default:
		applied = false
	}

	if applied {
		o.Version++
	}
}

func (o *Order) RaiseEvent(event DomainEvent) {
	o.Apply(event)
	o.UncommittedEvents = append(o.UncommittedEvents, event)
}

func (o *Order) LoadFromHistory(events []DomainEvent) {
	for _, event := range events {
		o.Apply(event)
	}
	o.ClearUncommittedEvents()
}

func (o *Order) ClearUncommittedEvents() {
	o.UncommittedEvents = nil
}

func (o *Order) applyOrderCreated(event OrderCreatedEvent) {
	o.ID = event.OrderID
	o.CustomerID = event.CustomerID
	o.Status = OrderStatusCreated
	o.Items = cloneOrderItems(event.Items)
	o.TotalAmount = float64(event.Amount)
}

func calculateTotalAmount(items []OrderItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Price * int64(item.Quantity)
	}
	return total
}

func cloneOrderItems(items []OrderItem) []OrderItem {
	if len(items) == 0 {
		return nil
	}

	cloned := make([]OrderItem, len(items))
	copy(cloned, items)
	return cloned
}
