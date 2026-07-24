package clients

import (
	"context"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/clients/inventorypb"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InventoryClient struct {
	conn   *grpc.ClientConn
	client inventorypb.InventoryServiceClient
}

func NewInventoryClient(target string) (*InventoryClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &InventoryClient{
		conn:   conn,
		client: inventorypb.NewInventoryServiceClient(conn),
	}, nil
}

func (c *InventoryClient) ReserveInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	response, err := c.client.ReserveInventory(ctx, &inventorypb.ReserveInventoryRequest{
		OrderId: orderID,
		Items:   mapInventoryItems(items),
	})
	if err != nil {
		return err
	}

	return ensureStatus(response.GetStatus(), response.GetReason(), "RESERVED")
}

func (c *InventoryClient) ReleaseInventory(ctx context.Context, orderID string, items []domain.OrderItem) error {
	response, err := c.client.CancelReservation(ctx, &inventorypb.CancelReservationRequest{
		OrderId: orderID,
		Items:   mapInventoryItems(items),
	})
	if err != nil {
		return err
	}

	return ensureStatus(response.GetStatus(), response.GetReason(), "CANCELLED")
}

func (c *InventoryClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func mapInventoryItems(items []domain.OrderItem) []*inventorypb.InventoryItem {
	if len(items) == 0 {
		return nil
	}

	mapped := make([]*inventorypb.InventoryItem, len(items))
	for i, item := range items {
		mapped[i] = &inventorypb.InventoryItem{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
		}
	}

	return mapped
}

var _ domain.InventoryService = (*InventoryClient)(nil)
