package service

import (
	"context"
	"errors"

	"github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/domain"
)

type Repository interface {
	ListProducts(ctx context.Context) ([]domain.Product, error)
	ReserveStock(ctx context.Context, orderID string, lines []domain.ReservationLine) ([]domain.ReservedItem, int64, error)
	ReleaseReservation(ctx context.Context, orderID string) error
	ConfirmReservation(ctx context.Context, orderID string) error
}

type ProductCache interface {
	GetProducts(ctx context.Context) ([]domain.Product, bool, error)
	SetProducts(ctx context.Context, products []domain.Product) error
	DeleteProducts(ctx context.Context) error
}

type Service struct {
	repo  Repository
	cache ProductCache
}

func New(repo Repository, cache ProductCache) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) ListProducts(ctx context.Context) ([]domain.Product, error) {
	if s.cache != nil {
		products, ok, err := s.cache.GetProducts(ctx)
		if err == nil && ok {
			return products, nil
		}
	}

	products, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetProducts(ctx, products)
	}
	return products, nil
}

func (s *Service) ReserveStock(ctx context.Context, orderID string, lines []domain.ReservationLine) ([]domain.ReservedItem, int64, error) {
	if orderID == "" {
		return nil, 0, errors.New("order id is required")
	}
	if len(lines) == 0 {
		return nil, 0, errors.New("at least one item is required")
	}
	for _, line := range lines {
		if line.ProductID == "" || line.Quantity <= 0 {
			return nil, 0, errors.New("each item needs a product id and positive quantity")
		}
	}
	items, total, err := s.repo.ReserveStock(ctx, orderID, lines)
	if err != nil {
		return nil, 0, err
	}
	s.deleteProductCache(ctx)
	return items, total, nil
}

func (s *Service) ReleaseReservation(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("order id is required")
	}
	if err := s.repo.ReleaseReservation(ctx, orderID); err != nil {
		return err
	}
	s.deleteProductCache(ctx)
	return nil
}

func (s *Service) ConfirmReservation(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("order id is required")
	}
	return s.repo.ConfirmReservation(ctx, orderID)
}

func (s *Service) deleteProductCache(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.DeleteProducts(ctx)
	}
}
