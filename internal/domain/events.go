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
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderCreatedEvent struct {
	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
	Amount     float64     `json:"amount"`
	Currency   string      `json:"currency,omitempty"`
	OccurredAt time.Time   `json:"occurred_at"`
}

func (OrderCreatedEvent) EventType() string {
	return EventTypeOrderCreated
}

type PaymentAuthorizedEvent struct {
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id,omitempty"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (PaymentAuthorizedEvent) EventType() string {
	return EventTypePaymentAuthorized
}

type PaymentFailedEvent struct {
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id,omitempty"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (PaymentFailedEvent) EventType() string {
	return EventTypePaymentFailed
}

type InventoryReservedEvent struct {
	OrderID       string      `json:"order_id"`
	ReservationID string      `json:"reservation_id,omitempty"`
	Items         []OrderItem `json:"items"`
	OccurredAt    time.Time   `json:"occurred_at"`
}

func (InventoryReservedEvent) EventType() string {
	return EventTypeInventoryReserved
}

type InventoryFailedEvent struct {
	OrderID    string      `json:"order_id"`
	Items      []OrderItem `json:"items"`
	Reason     string      `json:"reason"`
	OccurredAt time.Time   `json:"occurred_at"`
}

func (InventoryFailedEvent) EventType() string {
	return EventTypeInventoryFailed
}

type OrderCompletedEvent struct {
	OrderID    string    `json:"order_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (OrderCompletedEvent) EventType() string {
	return EventTypeOrderCompleted
}

type OrderCancelledEvent struct {
	OrderID    string    `json:"order_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (OrderCancelledEvent) EventType() string {
	return EventTypeOrderCancelled
}
