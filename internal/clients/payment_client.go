package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/clients/paymentpb"
	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	conn   *grpc.ClientConn
	client paymentpb.PaymentServiceClient
}

func NewPaymentClient(target string) (*PaymentClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &PaymentClient{
		conn:   conn,
		client: paymentpb.NewPaymentServiceClient(conn),
	}, nil
}

func (c *PaymentClient) ProcessPayment(ctx context.Context, orderID string, amount float64) error {
	response, err := c.client.AuthorizePayment(ctx, &paymentpb.AuthorizePaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return err
	}

	return ensureStatus(response.GetStatus(), response.GetReason(), "AUTHORIZED")
}

func (c *PaymentClient) RefundPayment(ctx context.Context, orderID string, amount float64) error {
	response, err := c.client.RefundPayment(ctx, &paymentpb.RefundPaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return err
	}

	return ensureStatus(response.GetStatus(), response.GetReason(), "REFUNDED")
}

func (c *PaymentClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func ensureStatus(status, reason, expected string) error {
	normalizedStatus := strings.ToUpper(status)
	if normalizedStatus == expected || normalizedStatus == "OK" || normalizedStatus == "SUCCESS" {
		return nil
	}
	if reason == "" {
		reason = fmt.Sprintf("unexpected status %q", status)
	}
	return fmt.Errorf("%s failed: %s", strings.ToLower(expected), reason)
}

var _ domain.PaymentService = (*PaymentClient)(nil)
