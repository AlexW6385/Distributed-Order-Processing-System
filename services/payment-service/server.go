package main

import (
	"context"
	"errors"
	"net"
	"time"

	paymentv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/payment/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	service *payment.Service
}

func newPaymentServer(service *payment.Service) *paymentServer {
	return &paymentServer{service: service}
}

func (s *paymentServer) PayOrder(ctx context.Context, request *paymentv1.PayOrderRequest) (*paymentv1.PayOrderResponse, error) {
	createdPayment, err := s.service.PayOrder(ctx, payment.PayOrderRequest{
		OrderID:        request.GetOrderId(),
		IdempotencyKey: request.GetIdempotencyKey(),
		AmountCents:    int(request.GetAmountCents()),
	})
	if err != nil {
		switch {
		case errors.Is(err, payment.ErrInvalidInput):
			return nil, status.Error(codes.InvalidArgument, "invalid payment request")
		case errors.Is(err, payment.ErrConflict):
			return nil, status.Error(codes.AlreadyExists, "payment already exists or idempotency key was used")
		default:
			return nil, status.Error(codes.Internal, "failed to pay order")
		}
	}

	return &paymentv1.PayOrderResponse{
		Payment: toProtoPayment(createdPayment),
	}, nil
}

func listenGRPC(port string) (net.Listener, error) {
	return net.Listen("tcp", ":"+port)
}

func toProtoPayment(payment payment.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		Id:             payment.ID,
		OrderId:        payment.OrderID,
		IdempotencyKey: payment.IdempotencyKey,
		Status:         payment.Status,
		AmountCents:    int32(payment.AmountCents),
		CreatedAt:      payment.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:      payment.UpdatedAt.Format(time.RFC3339Nano),
	}
}
