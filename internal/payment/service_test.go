package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	createRequest PayOrderRequest
	createPayment Payment
	createErr     error
	createCalls   int
}

func (s *fakeStore) Create(ctx context.Context, request PayOrderRequest) (Payment, error) {
	s.createCalls++
	s.createRequest = request
	return s.createPayment, s.createErr
}

type fakeIdempotencyStore struct {
	reserved     bool
	reserveKey   string
	reserveErr   error
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

func TestServicePayOrderValidatesInput(t *testing.T) {
	service := NewService(&fakeStore{}, &fakeIdempotencyStore{reserved: true})

	tests := []struct {
		name    string
		request PayOrderRequest
	}{
		{name: "missing order id", request: PayOrderRequest{IdempotencyKey: "payment-1", AmountCents: 1000}},
		{name: "missing idempotency key", request: PayOrderRequest{OrderID: "order-1", AmountCents: 1000}},
		{name: "invalid amount", request: PayOrderRequest{OrderID: "order-1", IdempotencyKey: "payment-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.PayOrder(context.Background(), tt.request)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestServicePayOrderReservesIdempotencyKey(t *testing.T) {
	store := &fakeStore{createPayment: Payment{ID: "payment-1"}}
	idempotencyStore := &fakeIdempotencyStore{reserved: true}
	service := NewService(store, idempotencyStore)

	_, err := service.PayOrder(context.Background(), PayOrderRequest{
		OrderID:        " order-1 ",
		IdempotencyKey: " payment-1 ",
		AmountCents:    12999,
	})
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if idempotencyStore.reserveKey != "payment-1" {
		t.Fatalf("expected reserve key payment-1, got %q", idempotencyStore.reserveKey)
	}
	if store.createRequest.OrderID != "order-1" {
		t.Fatalf("expected trimmed order id, got %q", store.createRequest.OrderID)
	}
}

func TestServicePayOrderRejectsDuplicateIdempotencyKey(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, &fakeIdempotencyStore{reserved: false})

	_, err := service.PayOrder(context.Background(), PayOrderRequest{
		OrderID:        "order-1",
		IdempotencyKey: "payment-1",
		AmountCents:    12999,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if store.createCalls != 0 {
		t.Fatalf("expected store not to be called, got %d calls", store.createCalls)
	}
}
