package infrastructure

import (
	"context"
	"errors"
	"sync"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
)

var ErrConcurrencyConflict = errors.New("concurrency conflict")

type MemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]domain.DomainEvent
}

func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events: make(map[string][]domain.DomainEvent),
	}
}

func (s *MemoryEventStore) SaveEvents(ctx context.Context, aggregateID string, events []domain.DomainEvent, expectedVersion int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.events == nil {
		s.events = make(map[string][]domain.DomainEvent)
	}

	currentVersion := len(s.events[aggregateID])
	if expectedVersion != currentVersion {
		return ErrConcurrencyConflict
	}

	s.events[aggregateID] = append(s.events[aggregateID], events...)
	return nil
}

func (s *MemoryEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]domain.DomainEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[aggregateID]
	if len(events) == 0 {
		return nil, nil
	}

	loaded := make([]domain.DomainEvent, len(events))
	copy(loaded, events)
	return loaded, nil
}

var _ domain.EventStore = (*MemoryEventStore)(nil)
