package order

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, request CreateOrderRequest) (Order, error) {
	request.CustomerEmail = strings.TrimSpace(request.CustomerEmail)
	if request.CustomerEmail == "" || len(request.Items) == 0 {
		return Order{}, ErrInvalidInput
	}

	for i := range request.Items {
		request.Items[i].ProductID = strings.TrimSpace(request.Items[i].ProductID)
		if request.Items[i].ProductID == "" || request.Items[i].Quantity <= 0 {
			return Order{}, ErrInvalidInput
		}
	}

	return s.repository.Create(ctx, request)
}

func (s *Service) Get(ctx context.Context, orderID string) (Order, error) {
	foundOrder, err := s.repository.Find(ctx, strings.TrimSpace(orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return foundOrder, err
}

func (s *Service) Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error) {
	orderID = strings.TrimSpace(orderID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if orderID == "" || request.IdempotencyKey == "" {
		return PaidOrder{}, ErrInvalidInput
	}

	return s.repository.Pay(ctx, orderID, request)
}
