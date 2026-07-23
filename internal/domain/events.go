package domain

import "time"

const (
	EventTypeOrderCreated      = "order.created"
	EventTypePaymentAuthorized = "payment.authorized"
	EventTypePaymentFailed     = "payment.failed"
	EventTypeInventoryReserved = "inventory.reserved"
	EventTypeInventoryFailed   = "inventory.failed"
	EventTypeOrderCompleted    = "order.completed"
	EventTypeOrderCancelled    = "order.cancelled"
)

type DomainEvent interface {
	EventType() string
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     int64
}

type OrderCreatedEvent struct {
	OrderID    string
	CustomerID string
	Items      []OrderItem
	Amount     int64
	Currency   string
	OccurredAt time.Time
}

func (OrderCreatedEvent) EventType() string {
	return EventTypeOrderCreated
}

type PaymentAuthorizedEvent struct {
	OrderID    string
	PaymentID  string
	Amount     int64
	Currency   string
	OccurredAt time.Time
}

func (PaymentAuthorizedEvent) EventType() string {
	return EventTypePaymentAuthorized
}

type PaymentFailedEvent struct {
	OrderID    string
	PaymentID  string
	Reason     string
	OccurredAt time.Time
}

func (PaymentFailedEvent) EventType() string {
	return EventTypePaymentFailed
}

type InventoryReservedEvent struct {
	OrderID       string
	ReservationID string
	Items         []OrderItem
	OccurredAt    time.Time
}

func (InventoryReservedEvent) EventType() string {
	return EventTypeInventoryReserved
}

type InventoryFailedEvent struct {
	OrderID    string
	Items      []OrderItem
	Reason     string
	OccurredAt time.Time
}

func (InventoryFailedEvent) EventType() string {
	return EventTypeInventoryFailed
}

type OrderCompletedEvent struct {
	OrderID    string
	OccurredAt time.Time
}

func (OrderCompletedEvent) EventType() string {
	return EventTypeOrderCompleted
}

type OrderCancelledEvent struct {
	OrderID    string
	Reason     string
	OccurredAt time.Time
}

func (OrderCancelledEvent) EventType() string {
	return EventTypeOrderCancelled
}
