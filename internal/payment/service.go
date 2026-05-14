package payment

import (
	"context"
	"strings"
	"time"
)

type Store interface {
	Create(ctx context.Context, request PayOrderRequest) (Payment, error)
}

type Service struct {
	repository       Store
	idempotencyStore IdempotencyStore
}

func NewService(repository Store, idempotencyStore IdempotencyStore) *Service {
	return &Service{
		repository:       repository,
		idempotencyStore: idempotencyStore,
	}
}

func (s *Service) PayOrder(ctx context.Context, request PayOrderRequest) (Payment, error) {
	request.OrderID = strings.TrimSpace(request.OrderID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.OrderID == "" || request.IdempotencyKey == "" || request.AmountCents <= 0 {
		return Payment{}, ErrInvalidInput
	}

	reserved, err := s.idempotencyStore.ReservePayment(ctx, request.IdempotencyKey, 24*time.Hour)
	if err != nil {
		return Payment{}, err
	}
	if !reserved {
		return Payment{}, ErrConflict
	}

	createdPayment, err := s.repository.Create(ctx, request)
	if err != nil {
		return Payment{}, err
	}

	return createdPayment, nil
}
