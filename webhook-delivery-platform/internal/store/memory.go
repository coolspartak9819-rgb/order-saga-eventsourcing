package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/domain"
)

var ErrNotFound = errors.New("delivery not found")

type Queue struct {
	mu          sync.RWMutex
	deliveries  map[string]*domain.Delivery
	byEvent     map[string]string
	ready       chan string
	deadLetters []string
}

func NewMemoryQueue() *Queue {
	return &Queue{
		deliveries: make(map[string]*domain.Delivery),
		byEvent:    make(map[string]string),
		ready:      make(chan string, 1024),
	}
}

func (q *Queue) CreateOrGet(event domain.Event, delivery domain.Delivery) (domain.Delivery, bool) {
	key := event.TenantID + ":" + event.ID
	q.mu.Lock()
	defer q.mu.Unlock()
	if existingID, ok := q.byEvent[key]; ok {
		return *q.deliveries[existingID], true
	}
	q.deliveries[delivery.ID] = &delivery
	q.byEvent[key] = delivery.ID
	return delivery, false
}

func (q *Queue) Get(id string) (domain.Delivery, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	delivery, ok := q.deliveries[id]
	if !ok {
		return domain.Delivery{}, ErrNotFound
	}
	return *delivery, nil
}

func (q *Queue) Enqueue(ctx context.Context, id string) error {
	select {
	case q.ready <- id:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue) Next(ctx context.Context) (string, error) {
	select {
	case id := <-q.ready:
		return id, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (q *Queue) Update(id string, update func(*domain.Delivery)) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delivery, ok := q.deliveries[id]
	if !ok {
		return ErrNotFound
	}
	update(delivery)
	return nil
}

func (q *Queue) AddDeadLetter(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.deadLetters = append(q.deadLetters, id)
}

func (q *Queue) DeadLetters() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return append([]string(nil), q.deadLetters...)
}

func Due(now, next time.Time) bool {
	return !next.After(now)
}
