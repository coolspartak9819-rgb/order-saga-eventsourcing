package domain

import (
	"context"
	"time"
)

type EventStore interface {
	SaveEvents(ctx context.Context, aggregateID string, events []DomainEvent, expectedVersion int) error
	LoadEvents(ctx context.Context, aggregateID string) ([]DomainEvent, error)
}

type EventEnvelope struct {
	AggregateID string
	EventType   string
	Payload     []byte
	Version     int
	CreatedAt   time.Time
}
