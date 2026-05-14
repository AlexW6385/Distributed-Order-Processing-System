package order

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	createRequest CreateOrderRequest
	findOrderID   string
	payOrderID    string
	payRequest    PayOrderRequest
	payCalls      int

	createOrder Order
	createErr   error
	findOrder   Order
	findErr     error
	payResult   PaidOrder
	payErr      error
}

func (s *fakeStore) Create(ctx context.Context, request CreateOrderRequest) (Order, error) {
	s.createRequest = request
	return s.createOrder, s.createErr
}

func (s *fakeStore) Find(ctx context.Context, orderID string) (Order, error) {
	s.findOrderID = orderID
	return s.findOrder, s.findErr
}

func (s *fakeStore) Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error) {
	s.payCalls++
	s.payOrderID = orderID
	s.payRequest = request
	return s.payResult, s.payErr
}

type fakeIdempotencyStore struct {
	reserved     bool
	reserveErr   error
	reserveKey   string
	releasedKey  string
	releaseCalls int
}

func (s *fakeIdempotencyStore) ReservePayment(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	s.reserveKey = key
	return s.reserved, s.reserveErr
}

func (s *fakeIdempotencyStore) ReleasePayment(ctx context.Context, key string) error {
	s.releaseCalls++
	s.releasedKey = key
	return nil
}

type fakeProductCache struct {
	deleteCalls int
}

func (c *fakeProductCache) DeleteProducts(ctx context.Context) error {
	c.deleteCalls++
	return nil
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
	service := NewService(store)

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
}

func TestServiceCreateInvalidatesProductCache(t *testing.T) {
	store := &fakeStore{createOrder: Order{ID: "order-1"}}
	productCache := &fakeProductCache{}
	service := NewService(store, WithProductCache(productCache))

	_, err := service.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items:         []CreateOrderItemRequest{{ProductID: "product-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if productCache.deleteCalls != 1 {
		t.Fatalf("expected product cache invalidation once, got %d", productCache.deleteCalls)
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

func TestServicePayTrimsInputBeforeCallingStore(t *testing.T) {
	store := &fakeStore{payResult: PaidOrder{Order: Order{ID: "order-1"}}}
	service := NewService(store)

	_, err := service.Pay(context.Background(), " order-1 ", PayOrderRequest{IdempotencyKey: " payment-1 "})
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if store.payOrderID != "order-1" {
		t.Fatalf("expected trimmed order id, got %q", store.payOrderID)
	}
	if store.payRequest.IdempotencyKey != "payment-1" {
		t.Fatalf("expected trimmed idempotency key, got %q", store.payRequest.IdempotencyKey)
	}
}

func TestServicePayUsesIdempotencyReservation(t *testing.T) {
	store := &fakeStore{payResult: PaidOrder{Order: Order{ID: "order-1"}}}
	idempotencyStore := &fakeIdempotencyStore{reserved: true}
	service := NewService(store, WithIdempotencyStore(idempotencyStore))

	_, err := service.Pay(context.Background(), "order-1", PayOrderRequest{IdempotencyKey: "payment-1"})
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if idempotencyStore.reserveKey != "payment-1" {
		t.Fatalf("expected reserve key payment-1, got %q", idempotencyStore.reserveKey)
	}
	if store.payCalls != 1 {
		t.Fatalf("expected repository pay once, got %d", store.payCalls)
	}
}

func TestServicePayRejectsDuplicateIdempotencyKey(t *testing.T) {
	store := &fakeStore{}
	idempotencyStore := &fakeIdempotencyStore{reserved: false}
	service := NewService(store, WithIdempotencyStore(idempotencyStore))

	_, err := service.Pay(context.Background(), "order-1", PayOrderRequest{IdempotencyKey: "payment-1"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if store.payCalls != 0 {
		t.Fatalf("expected repository pay not to be called, got %d calls", store.payCalls)
	}
}

func TestServicePayReleasesReservationOnRepositoryFailure(t *testing.T) {
	store := &fakeStore{payErr: ErrNotFound}
	idempotencyStore := &fakeIdempotencyStore{reserved: true}
	service := NewService(store, WithIdempotencyStore(idempotencyStore))

	_, err := service.Pay(context.Background(), "order-1", PayOrderRequest{IdempotencyKey: "payment-1"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if idempotencyStore.releaseCalls != 1 {
		t.Fatalf("expected reservation release once, got %d", idempotencyStore.releaseCalls)
	}
	if idempotencyStore.releasedKey != "payment-1" {
		t.Fatalf("expected released key payment-1, got %q", idempotencyStore.releasedKey)
	}
}
