package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/domain"
	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/store"
)

const maxAttempts = 5

type DeliveryService struct {
	queue     *store.Queue
	client    *http.Client
	logger    *slog.Logger
	accepted  atomic.Uint64
	delivered atomic.Uint64
	retried   atomic.Uint64
	dead      atomic.Uint64
}

func NewDeliveryService(queue *store.Queue, client *http.Client, logger *slog.Logger) *DeliveryService {
	return &DeliveryService{queue: queue, client: client, logger: logger}
}

func (s *DeliveryService) Accept(ctx context.Context, event domain.Event, endpointURL string) (domain.Delivery, bool, error) {
	if event.ID == "" || event.TenantID == "" || event.EndpointID == "" || event.Type == "" {
		return domain.Delivery{}, false, fmt.Errorf("event_id, tenant_id, endpoint_id and type are required")
	}
	if endpointURL == "" {
		return domain.Delivery{}, false, fmt.Errorf("endpoint_url is required")
	}
	now := time.Now().UTC()
	delivery := domain.Delivery{
		ID: "dlv_" + event.ID, EventID: event.ID, TenantID: event.TenantID,
		EndpointID: event.EndpointID, EndpointURL: endpointURL, EventType: event.Type, Payload: event.Payload,
		Status: domain.StatusPending, NextAttempt: now, CreatedAt: now,
	}
	result, duplicate := s.queue.CreateOrGet(event, delivery)
	if duplicate {
		return result, true, nil
	}
	s.accepted.Add(1)
	if err := s.queue.Enqueue(ctx, result.ID); err != nil {
		return domain.Delivery{}, false, err
	}
	return result, false, nil
}

func (s *DeliveryService) Get(id string) (domain.Delivery, error) { return s.queue.Get(id) }

func (s *DeliveryService) Metrics() string {
	return strings.Join([]string{
		"# HELP webhook_events_accepted_total Incoming events accepted by the API.",
		"# TYPE webhook_events_accepted_total counter",
		"webhook_events_accepted_total " + strconv.FormatUint(s.accepted.Load(), 10),
		"# HELP webhook_deliveries_delivered_total Successful webhook deliveries.",
		"# TYPE webhook_deliveries_delivered_total counter",
		"webhook_deliveries_delivered_total " + strconv.FormatUint(s.delivered.Load(), 10),
		"# HELP webhook_delivery_retries_total Delivery attempts that required a retry.",
		"# TYPE webhook_delivery_retries_total counter",
		"webhook_delivery_retries_total " + strconv.FormatUint(s.retried.Load(), 10),
		"# HELP webhook_dead_letters_total Deliveries moved to the dead-letter queue.",
		"# TYPE webhook_dead_letters_total counter",
		"webhook_dead_letters_total " + strconv.FormatUint(s.dead.Load(), 10),
		"",
	}, "\n")
}

func (s *DeliveryService) Run(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go s.worker(ctx)
	}
}

func (s *DeliveryService) worker(ctx context.Context) {
	for {
		id, err := s.queue.Next(ctx)
		if err != nil {
			return
		}
		if err := s.deliver(ctx, id); err != nil {
			s.logger.Error("delivery failed", "delivery_id", id, "error", err)
		}
	}
}

func (s *DeliveryService) deliver(ctx context.Context, id string) error {
	delivery, err := s.queue.Get(id)
	if err != nil {
		return err
	}
	if delivery.Status == domain.StatusDelivered || delivery.Status == domain.StatusDead {
		return nil
	}
	// The memory adapter keeps the sample self-contained. The PostgreSQL adapter will persist this envelope.
	payload, _ := json.Marshal(map[string]any{
		"delivery_id": delivery.ID, "event_id": delivery.EventID, "tenant_id": delivery.TenantID,
		"type": delivery.EventType, "payload": delivery.Payload,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.EndpointURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.delivered.Add(1)
			return s.queue.Update(id, func(item *domain.Delivery) {
				item.Status = domain.StatusDelivered
				item.DeliveredAt = time.Now().UTC()
				item.Attempts++
			})
		}
		err = fmt.Errorf("endpoint returned %s", resp.Status)
	}
	s.retried.Add(1)
	return s.retryOrDeadLetter(ctx, id, err)
}

func (s *DeliveryService) retryOrDeadLetter(ctx context.Context, id string, deliveryErr error) error {
	var dead bool
	var retryAt time.Time
	err := s.queue.Update(id, func(item *domain.Delivery) {
		item.Attempts++
		item.LastError = deliveryErr.Error()
		if item.Attempts >= maxAttempts {
			item.Status = domain.StatusDead
			dead = true
			return
		}
		item.Status = domain.StatusRetrying
		item.NextAttempt = time.Now().UTC().Add(time.Duration(1<<min(item.Attempts, 4)) * time.Second)
		retryAt = item.NextAttempt
	})
	if dead {
		s.dead.Add(1)
		s.queue.AddDeadLetter(id)
	} else if !retryAt.IsZero() {
		go func() {
			timer := time.NewTimer(time.Until(retryAt))
			defer timer.Stop()
			select {
			case <-timer.C:
				_ = s.queue.Enqueue(ctx, id)
			case <-ctx.Done():
			}
		}()
	}
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
