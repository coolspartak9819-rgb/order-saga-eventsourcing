package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/domain"
	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/store"
)

func TestAcceptIsIdempotentPerTenantAndEvent(t *testing.T) {
	q := store.NewMemoryQueue()
	s := NewDeliveryService(q, http.DefaultClient, slog.Default())
	event := domain.Event{ID: "evt_1", TenantID: "tenant_a", EndpointID: "endpoint_1", Type: "invoice.created"}
	first, duplicate, err := s.Accept(context.Background(), event, "https://example.test/webhook")
	if err != nil || duplicate {
		t.Fatalf("first accept: delivery=%+v duplicate=%v err=%v", first, duplicate, err)
	}
	second, duplicate, err := s.Accept(context.Background(), event, "https://example.test/webhook")
	if err != nil || !duplicate || first.ID != second.ID {
		t.Fatalf("second accept: delivery=%+v duplicate=%v err=%v", second, duplicate, err)
	}
}

func TestDue(t *testing.T) {
	now := time.Now()
	if store.Due(now, now.Add(time.Second)) {
		t.Fatal("future delivery must not be due")
	}
	if !store.Due(now, now.Add(-time.Second)) {
		t.Fatal("past delivery must be due")
	}
}

func TestWorkerDeliversEventPayload(t *testing.T) {
	received := make(chan bool, 1)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || !strings.Contains(string(body), "invoice.created") {
			t.Errorf("unexpected webhook request: %s %s %s", r.Method, r.Header.Get("Content-Type"), body)
		}
		received <- true
		return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	})}

	q := store.NewMemoryQueue()
	s := NewDeliveryService(q, client, slog.Default())
	s.Run(context.Background(), 1)
	event := domain.Event{ID: "evt_2", TenantID: "tenant_a", EndpointID: "endpoint_1", Type: "invoice.created", Payload: map[string]any{"amount": 1200}}
	delivery, _, err := s.Accept(context.Background(), event, "https://endpoint.test/webhook")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("webhook was not delivered")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, getErr := s.Get(delivery.ID)
		if getErr == nil && current.Status == domain.StatusDelivered {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	current, _ := s.Get(delivery.ID)
	t.Fatalf("delivery did not reach delivered status: %+v", current)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
