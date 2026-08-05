package domain

import "time"

type Event struct {
	ID         string         `json:"event_id"`
	TenantID   string         `json:"tenant_id"`
	EndpointID string         `json:"endpoint_id"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
}

type Delivery struct {
	ID          string         `json:"delivery_id"`
	EventID     string         `json:"event_id"`
	TenantID    string         `json:"tenant_id"`
	EndpointID  string         `json:"endpoint_id"`
	EndpointURL string         `json:"-"`
	EventType   string         `json:"event_type"`
	Payload     map[string]any `json:"payload"`
	Status      string         `json:"status"`
	Attempts    int            `json:"attempts"`
	NextAttempt time.Time      `json:"next_attempt_at"`
	LastError   string         `json:"last_error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DeliveredAt time.Time      `json:"delivered_at,omitempty"`
}

const (
	StatusPending   = "pending"
	StatusDelivered = "delivered"
	StatusRetrying  = "retrying"
	StatusDead      = "dead_letter"
)
