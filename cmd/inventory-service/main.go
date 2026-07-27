package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/clients/inventorypb"
	"google.golang.org/grpc"
)

const inventoryServiceAddr = ":50052"

type inventoryServer struct {
	inventorypb.UnimplementedInventoryServiceServer

	mu           sync.Mutex
	reservations map[string][]*inventorypb.InventoryItem
}

func newInventoryServer() *inventoryServer {
	return &inventoryServer{
		reservations: make(map[string][]*inventorypb.InventoryItem),
	}
}

func (s *inventoryServer) ReserveInventory(ctx context.Context, req *inventorypb.ReserveInventoryRequest) (*inventorypb.InventoryResponse, error) {
	if failureEnabled("INVENTORY_FAIL") {
		return inventoryResponse(req.GetOrderId(), req.GetItems(), "FAILED", "inventory failure injection is enabled"), nil
	}

	if len(req.GetItems()) == 0 {
		return inventoryResponse(req.GetOrderId(), req.GetItems(), "FAILED", "items are required"), nil
	}
	for _, item := range req.GetItems() {
		if item.GetProductId() == "" || item.GetQuantity() <= 0 {
			return inventoryResponse(req.GetOrderId(), req.GetItems(), "FAILED", "invalid inventory item"), nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.reservations[req.GetOrderId()] = cloneItems(req.GetItems())
	return inventoryResponse(req.GetOrderId(), req.GetItems(), "RESERVED", ""), nil
}

func failureEnabled(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func (s *inventoryServer) CancelReservation(ctx context.Context, req *inventorypb.CancelReservationRequest) (*inventorypb.InventoryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.reservations[req.GetOrderId()]; !ok {
		return inventoryResponse(req.GetOrderId(), req.GetItems(), "CANCELLED", "reservation not found"), nil
	}

	delete(s.reservations, req.GetOrderId())
	return inventoryResponse(req.GetOrderId(), req.GetItems(), "CANCELLED", ""), nil
}

func main() {
	listener, err := net.Listen("tcp", inventoryServiceAddr)
	if err != nil {
		log.Fatalf("listen inventory service: %v", err)
	}

	server := grpc.NewServer()
	inventorypb.RegisterInventoryServiceServer(server, newInventoryServer())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("inventory service listening on %s", inventoryServiceAddr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("serve inventory service: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	log.Println("shutting down inventory service")
	server.GracefulStop()
	log.Println("inventory service stopped")
}

func inventoryResponse(orderID string, items []*inventorypb.InventoryItem, status, reason string) *inventorypb.InventoryResponse {
	return &inventorypb.InventoryResponse{
		OrderId: orderID,
		Items:   cloneItems(items),
		Status:  status,
		Reason:  reason,
	}
}

func cloneItems(items []*inventorypb.InventoryItem) []*inventorypb.InventoryItem {
	if len(items) == 0 {
		return nil
	}

	cloned := make([]*inventorypb.InventoryItem, len(items))
	for i, item := range items {
		cloned[i] = &inventorypb.InventoryItem{
			ProductId: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		}
	}

	return cloned
}
