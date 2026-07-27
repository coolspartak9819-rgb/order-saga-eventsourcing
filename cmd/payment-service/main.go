package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/clients/paymentpb"
	"google.golang.org/grpc"
)

const paymentServiceAddr = ":50051"

type paymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
}

func (s *paymentServer) AuthorizePayment(ctx context.Context, req *paymentpb.AuthorizePaymentRequest) (*paymentpb.PaymentResponse, error) {
	if failureEnabled("PAYMENT_FAIL") {
		return &paymentpb.PaymentResponse{
			OrderId: req.GetOrderId(),
			Amount:  req.GetAmount(),
			Status:  "FAILED",
			Reason:  "payment failure injection is enabled",
		}, nil
	}

	if req.GetAmount() <= 0 {
		return &paymentpb.PaymentResponse{
			OrderId: req.GetOrderId(),
			Amount:  req.GetAmount(),
			Status:  "FAILED",
			Reason:  "amount must be positive",
		}, nil
	}

	return &paymentpb.PaymentResponse{
		OrderId: req.GetOrderId(),
		Amount:  req.GetAmount(),
		Status:  "AUTHORIZED",
	}, nil
}

func failureEnabled(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func (s *paymentServer) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.PaymentResponse, error) {
	if req.GetAmount() <= 0 {
		return &paymentpb.PaymentResponse{
			OrderId: req.GetOrderId(),
			Amount:  req.GetAmount(),
			Status:  "FAILED",
			Reason:  "amount must be positive",
		}, nil
	}

	return &paymentpb.PaymentResponse{
		OrderId: req.GetOrderId(),
		Amount:  req.GetAmount(),
		Status:  "REFUNDED",
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", paymentServiceAddr)
	if err != nil {
		log.Fatalf("listen payment service: %v", err)
	}

	server := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(server, &paymentServer{})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("payment service listening on %s", paymentServiceAddr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("serve payment service: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	log.Println("shutting down payment service")
	server.GracefulStop()
	log.Println("payment service stopped")
}
