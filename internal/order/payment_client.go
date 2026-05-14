package order

import (
	"context"
	"time"

	paymentv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentGRPCClient struct {
	client paymentv1.PaymentServiceClient
}

func NewPaymentGRPCClient(client paymentv1.PaymentServiceClient) *PaymentGRPCClient {
	return &PaymentGRPCClient{client: client}
}

func (c *PaymentGRPCClient) PayOrder(ctx context.Context, orderID string, idempotencyKey string, amountCents int) (Payment, error) {
	response, err := c.client.PayOrder(ctx, &paymentv1.PayOrderRequest{
		OrderId:        orderID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    int32(amountCents),
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return Payment{}, ErrConflict
		}
		return Payment{}, err
	}

	return fromProtoPayment(response.GetPayment()), nil
}

func fromProtoPayment(item *paymentv1.Payment) Payment {
	createdAt, _ := time.Parse(time.RFC3339Nano, item.GetCreatedAt())
	updatedAt, _ := time.Parse(time.RFC3339Nano, item.GetUpdatedAt())

	return Payment{
		ID:             item.GetId(),
		OrderID:        item.GetOrderId(),
		IdempotencyKey: item.GetIdempotencyKey(),
		Status:         item.GetStatus(),
		AmountCents:    int(item.GetAmountCents()),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
