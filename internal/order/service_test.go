package order

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/events"
)

type fakeStore struct {
	createRequest  CreateOrderRequest
	createReserved []ReservedStock
	findOrderID    string
	payOrderID     string

	createOrder   Order
	createErr     error
	findOrder     Order
	findErr       error
	markPaid      Order
	markPaidErr   error
	markPaidEvent events.OrderPaidEvent
}

func (s *fakeStore) Create(ctx context.Context, request CreateOrderRequest, reservedItems []ReservedStock) (Order, error) {
	s.createRequest = request
	s.createReserved = reservedItems
	return s.createOrder, s.createErr
}

func (s *fakeStore) Find(ctx context.Context, orderID string) (Order, error) {
	s.findOrderID = orderID
	return s.findOrder, s.findErr
}

func (s *fakeStore) MarkPaid(ctx context.Context, orderID string, event events.OrderPaidEvent) (Order, error) {
	s.payOrderID = orderID
	s.markPaidEvent = event
	return s.markPaid, s.markPaidErr
}

type fakeProductClient struct {
	reserveCalls int
	reserveErr   error
}

func (c *fakeProductClient) ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error) {
	c.reserveCalls++
	return ReservedStock{
		ProductID:      productID,
		Quantity:       quantity,
		UnitPriceCents: 1000,
		SubtotalCents:  quantity * 1000,
	}, c.reserveErr
}

func TestServiceCreateValidatesInput(t *testing.T) {
	service := NewService(&fakeStore{})

	tests := []struct {
		name    string
		request CreateOrderRequest
	}{
		{
			name: "missing customer email",
			request: CreateOrderRequest{
				Items: []CreateOrderItemRequest{{ProductID: "product-1", Quantity: 1}},
			},
		},
		{
			name: "missing items",
			request: CreateOrderRequest{
				CustomerEmail: "alex@example.com",
			},
		},
		{
			name: "missing product id",
			request: CreateOrderRequest{
				CustomerEmail: "alex@example.com",
				Items:         []CreateOrderItemRequest{{Quantity: 1}},
			},
		},
		{
			name: "invalid quantity",
			request: CreateOrderRequest{
				CustomerEmail: "alex@example.com",
				Items:         []CreateOrderItemRequest{{ProductID: "product-1", Quantity: 0}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.request)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestServiceCreateTrimsInputBeforeCallingStore(t *testing.T) {
	store := &fakeStore{createOrder: Order{ID: "order-1"}}
	productClient := &fakeProductClient{}
	service := NewService(store, WithProductClient(productClient))

	_, err := service.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: " alex@example.com ",
		Items:         []CreateOrderItemRequest{{ProductID: " product-1 ", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if store.createRequest.CustomerEmail != "alex@example.com" {
		t.Fatalf("expected trimmed email, got %q", store.createRequest.CustomerEmail)
	}
	if store.createRequest.Items[0].ProductID != "product-1" {
		t.Fatalf("expected trimmed product id, got %q", store.createRequest.Items[0].ProductID)
	}
	if productClient.reserveCalls != 1 {
		t.Fatalf("expected product reservation once, got %d", productClient.reserveCalls)
	}
	if len(store.createReserved) != 1 || store.createReserved[0].SubtotalCents != 2000 {
		t.Fatalf("expected reserved item passed to store, got %+v", store.createReserved)
	}
}

func TestServiceGetMapsMissingOrder(t *testing.T) {
	service := NewService(&fakeStore{findErr: sql.ErrNoRows})

	_, err := service.Get(context.Background(), "missing-order")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServicePayValidatesInput(t *testing.T) {
	service := NewService(&fakeStore{})

	tests := []struct {
		name    string
		orderID string
		request PayOrderRequest
	}{
		{name: "missing order id", request: PayOrderRequest{IdempotencyKey: "payment-1"}},
		{name: "missing idempotency key", orderID: "order-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Pay(context.Background(), tt.orderID, tt.request)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

type fakePaymentClient struct {
	orderID        string
	idempotencyKey string
	amountCents    int
	payment        Payment
	err            error
}

func (c *fakePaymentClient) PayOrder(ctx context.Context, orderID string, idempotencyKey string, amountCents int) (Payment, error) {
	c.orderID = orderID
	c.idempotencyKey = idempotencyKey
	c.amountCents = amountCents
	return c.payment, c.err
}

func TestServicePayCallsPaymentClientAndMarksOrderPaid(t *testing.T) {
	store := &fakeStore{
		findOrder: Order{ID: "order-1", Status: "pending", TotalCents: 12999},
		markPaid:  Order{ID: "order-1", Status: "paid", TotalCents: 12999},
	}
	paymentClient := &fakePaymentClient{payment: Payment{ID: "payment-1"}}
	service := NewService(
		store,
		WithPaymentClient(paymentClient),
	)

	paidOrder, err := service.Pay(context.Background(), " order-1 ", PayOrderRequest{IdempotencyKey: " payment-1 "})
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if paymentClient.orderID != "order-1" {
		t.Fatalf("expected payment order id order-1, got %q", paymentClient.orderID)
	}
	if paymentClient.idempotencyKey != "payment-1" {
		t.Fatalf("expected idempotency key payment-1, got %q", paymentClient.idempotencyKey)
	}
	if paymentClient.amountCents != 12999 {
		t.Fatalf("expected payment amount 12999, got %d", paymentClient.amountCents)
	}
	if paidOrder.Order.Status != "paid" {
		t.Fatalf("expected paid order, got %q", paidOrder.Order.Status)
	}
	if store.markPaidEvent.OrderID != "order-1" {
		t.Fatalf("expected order paid event for order-1, got %+v", store.markPaidEvent)
	}
	if store.markPaidEvent.PaymentID != "payment-1" {
		t.Fatalf("expected payment id payment-1, got %q", store.markPaidEvent.PaymentID)
	}
}

func TestServicePayRejectsAlreadyPaidOrder(t *testing.T) {
	store := &fakeStore{findOrder: Order{ID: "order-1", Status: "paid", TotalCents: 12999}}
	service := NewService(store, WithPaymentClient(&fakePaymentClient{}))

	_, err := service.Pay(context.Background(), "order-1", PayOrderRequest{IdempotencyKey: "payment-1"})
	if !errors.Is(err, ErrAlreadyPaid) {
		t.Fatalf("expected ErrAlreadyPaid, got %v", err)
	}
}
