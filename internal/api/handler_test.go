package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/api"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/infrastructure"
)

func TestOrderHandlerCreateAndGetOrder(t *testing.T) {
	store := infrastructure.NewMemoryEventStore()
	saga := domain.NewOrderSagaOrchestrator(
		store,
		stubPaymentService{},
		stubInventoryService{},
	)
	handler := api.NewOrderHandler(store, saga)

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/orders",
		strings.NewReader(`{
			"customer_id":"customer-1",
			"items":[{"product_id":"product-1","quantity":2,"price":19.99}]
		}`),
	)
	createResponse := httptest.NewRecorder()

	handler.CreateOrder(createResponse, createRequest)

	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("CreateOrder() status = %d, want %d", createResponse.Code, http.StatusAccepted)
	}

	var created struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.OrderID == "" {
		t.Fatal("CreateOrder() returned an empty order id")
	}
	if created.Status != domain.OrderStatusCompleted {
		t.Fatalf("CreateOrder() status = %q, want %q", created.Status, domain.OrderStatusCompleted)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/orders/"+created.OrderID,
		nil,
	)
	getResponse := httptest.NewRecorder()

	handler.GetOrder(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GetOrder() status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	var restored struct {
		ID          string  `json:"id"`
		CustomerID  string  `json:"customer_id"`
		Status      string  `json:"status"`
		TotalAmount float64 `json:"total_amount"`
		Version     int     `json:"version"`
	}
	if err := json.Unmarshal(getResponse.Body.Bytes(), &restored); err != nil {
		t.Fatalf("decode get response: %v", err)
	}

	if restored.ID != created.OrderID {
		t.Fatalf("restored id = %q, want %q", restored.ID, created.OrderID)
	}
	if restored.CustomerID != "customer-1" {
		t.Fatalf("restored customer id = %q, want customer-1", restored.CustomerID)
	}
	if restored.Status != domain.OrderStatusCompleted {
		t.Fatalf("restored status = %q, want %q", restored.Status, domain.OrderStatusCompleted)
	}
	if restored.TotalAmount != 39.98 {
		t.Fatalf("restored total amount = %v, want 39.98", restored.TotalAmount)
	}
	if restored.Version != 4 {
		t.Fatalf("restored version = %d, want 4", restored.Version)
	}
}

type stubPaymentService struct{}

func (stubPaymentService) ProcessPayment(context.Context, string, float64) error {
	return nil
}

func (stubPaymentService) RefundPayment(context.Context, string, float64) error {
	return nil
}

type stubInventoryService struct{}

func (stubInventoryService) ReserveInventory(context.Context, string, []domain.OrderItem) error {
	return nil
}

func (stubInventoryService) ReleaseInventory(context.Context, string, []domain.OrderItem) error {
	return nil
}
