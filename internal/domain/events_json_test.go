package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
)

func TestOrderCreatedEventJSONUsesStableFieldNames(t *testing.T) {
	event := domain.OrderCreatedEvent{
		OrderID:    "order-1",
		CustomerID: "customer-1",
		Items: []domain.OrderItem{{
			ProductID: "product-1",
			Quantity:  2,
			Price:     19.99,
		}},
		Amount: 39.98,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded["order_id"] != "order-1" {
		t.Fatalf("order_id = %v, want order-1", decoded["order_id"])
	}
	if decoded["customer_id"] != "customer-1" {
		t.Fatalf("customer_id = %v, want customer-1", decoded["customer_id"])
	}
}
