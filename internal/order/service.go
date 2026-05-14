package order

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Service struct {
	repository       Store
	idempotencyStore IdempotencyStore
	productCache     ProductCache
}

type Store interface {
	Create(ctx context.Context, request CreateOrderRequest) (Order, error)
	Find(ctx context.Context, orderID string) (Order, error)
	Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error)
}

type ProductCache interface {
	DeleteProducts(ctx context.Context) error
}

type Option func(*Service)

func WithIdempotencyStore(store IdempotencyStore) Option {
	return func(service *Service) {
		service.idempotencyStore = store
	}
}

func WithProductCache(cache ProductCache) Option {
	return func(service *Service) {
		service.productCache = cache
	}
}

func NewService(repository Store, options ...Option) *Service {
	service := &Service{repository: repository}
	for _, option := range options {
		option(service)
	}
	return service
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

	createdOrder, err := s.repository.Create(ctx, request)
	if err != nil {
		return Order{}, err
	}

	if s.productCache != nil {
		_ = s.productCache.DeleteProducts(ctx)
	}

	return createdOrder, nil
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

	if s.idempotencyStore == nil {
		return s.repository.Pay(ctx, orderID, request)
	}

	reserved, err := s.idempotencyStore.ReservePayment(ctx, request.IdempotencyKey, 24*time.Hour)
	if err != nil {
		return PaidOrder{}, err
	}
	if !reserved {
		return PaidOrder{}, ErrConflict
	}

	paidOrder, err := s.repository.Pay(ctx, orderID, request)
	if err != nil {
		if !errors.Is(err, ErrAlreadyPaid) && !errors.Is(err, ErrConflict) {
			_ = s.idempotencyStore.ReleasePayment(ctx, request.IdempotencyKey)
		}
		return PaidOrder{}, err
	}

	return paidOrder, nil
}
