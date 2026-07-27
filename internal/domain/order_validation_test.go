package domain_test

import (
	"errors"
	"testing"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
)

func TestNewOrderRejectsInvalidItems(t *testing.T) {
	tests := []struct {
		name  string
		items []domain.OrderItem
	}{
		{
			name:  "missing product id",
			items: []domain.OrderItem{{Quantity: 1, Price: 10}},
		},
		{
			name:  "non-positive quantity",
			items: []domain.OrderItem{{ProductID: "product-1", Quantity: 0, Price: 10}},
		},
		{
			name:  "negative price",
			items: []domain.OrderItem{{ProductID: "product-1", Quantity: 1, Price: -1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewOrder("order-1", "customer-1", tt.items)
			if !errors.Is(err, domain.ErrInvalidOrderItem) {
				t.Fatalf("NewOrder() error = %v, want %v", err, domain.ErrInvalidOrderItem)
			}
		})
	}
}
