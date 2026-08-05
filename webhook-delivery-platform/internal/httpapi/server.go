package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/domain"
	"github.com/coolspartak9819-rgb/webhook-delivery-platform/internal/service"
)

type Server struct{ deliveries *service.DeliveryService }

func NewRouter(deliveries *service.DeliveryService) http.Handler {
	s := &Server{deliveries: deliveries}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /metrics", s.metrics)
	mux.HandleFunc("POST /v1/events", s.createEvent)
	mux.HandleFunc("GET /v1/deliveries/", s.getDelivery)
	return requestLogger(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.deliveries.Metrics()))
}

type createEventRequest struct {
	EventID     string         `json:"event_id"`
	EndpointID  string         `json:"endpoint_id"`
	EndpointURL string         `json:"endpoint_url"`
	Type        string         `json:"type"`
	Payload     map[string]any `json:"payload"`
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	var input createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	event := domain.Event{ID: input.EventID, TenantID: r.Header.Get("X-Tenant-ID"), EndpointID: input.EndpointID, Type: input.Type, Payload: input.Payload}
	delivery, duplicate, err := s.deliveries.Accept(r.Context(), event, input.EndpointURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := http.StatusAccepted
	if duplicate {
		status = http.StatusOK
	}
	w.Header().Set("Location", "/v1/deliveries/"+delivery.ID)
	writeJSON(w, status, map[string]any{"delivery": delivery, "duplicate": duplicate})
}

func (s *Server) getDelivery(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/deliveries/")
	delivery, err := s.deliveries.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	writeJSON(w, http.StatusOK, delivery)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
