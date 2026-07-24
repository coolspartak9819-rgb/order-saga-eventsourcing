package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
)

type OrderHandler struct {
	EventStore domain.EventStore
	Saga       *domain.OrderSagaOrchestrator
}

func NewOrderHandler(eventStore domain.EventStore, saga *domain.OrderSagaOrchestrator) *OrderHandler {
	return &OrderHandler{
		EventStore: eventStore,
		Saga:       saga,
	}
}

type createOrderRequest struct {
	CustomerID string             `json:"customer_id"`
	Items      []domain.OrderItem `json:"items"`
}

type createOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type getOrderResponse struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	Version     int     `json:"version"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	orderID, err := newUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate order id")
		return
	}

	order, err := domain.NewOrder(orderID, req.CustomerID, req.Items)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if h.Saga == nil {
		writeError(w, http.StatusInternalServerError, "order saga is not configured")
		return
	}
	if err := h.Saga.ExecuteOrderSaga(r.Context(), order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, createOrderResponse{
		OrderID: order.ID,
		Status:  order.Status,
	})
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := orderIDFromRequest(r)
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	if h.EventStore == nil {
		writeError(w, http.StatusInternalServerError, "event store is not configured")
		return
	}

	events, err := h.EventStore.LoadEvents(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	order := &domain.Order{}
	order.LoadFromHistory(events)

	writeJSON(w, http.StatusOK, getOrderResponse{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		Version:     order.Version,
	})
}

func orderIDFromRequest(r *http.Request) string {
	if orderID := r.PathValue("id"); orderID != "" {
		return orderID
	}
	if orderID := r.URL.Query().Get("order_id"); orderID != "" {
		return orderID
	}
	if orderID := r.URL.Query().Get("id"); orderID != "" {
		return orderID
	}

	const orderPathPrefix = "/api/orders/"
	if strings.HasPrefix(r.URL.Path, orderPathPrefix) {
		return strings.TrimPrefix(r.URL.Path, orderPathPrefix)
	}

	return ""
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{Error: message})
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	uuid := fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
	return uuid, nil
}
