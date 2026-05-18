package service

import (
	"context"
	"errors"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/domain"
)

func TestListProductsUsesCacheHit(t *testing.T) {
	ctx := context.Background()
	cached := []domain.Product{{ID: "prod-1", Name: "Coffee", PriceCents: 1299, Stock: 10}}
	repo := &fakeRepository{}
	cache := &fakeProductCache{products: cached, hit: true}

	products, err := New(repo, cache).ListProducts(ctx)
	if err != nil {
		t.Fatalf("ListProducts returned error: %v", err)
	}
	if repo.listCalls != 0 {
		t.Fatalf("expected cache hit to skip repository, got %d repo calls", repo.listCalls)
	}
	if len(products) != 1 || products[0].ID != cached[0].ID {
		t.Fatalf("unexpected products: %#v", products)
	}
}

func TestListProductsCachesRepositoryResult(t *testing.T) {
	ctx := context.Background()
	fromRepo := []domain.Product{{ID: "prod-2", Name: "Mug", PriceCents: 1899, Stock: 4}}
	repo := &fakeRepository{products: fromRepo}
	cache := &fakeProductCache{}

	products, err := New(repo, cache).ListProducts(ctx)
	if err != nil {
		t.Fatalf("ListProducts returned error: %v", err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected repository miss path, got %d repo calls", repo.listCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected repository result to be cached, got %d cache set calls", cache.setCalls)
	}
	if len(products) != 1 || products[0].ID != fromRepo[0].ID {
		t.Fatalf("unexpected products: %#v", products)
	}
}

func TestReserveStockInvalidatesProductCache(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepository{
		reserved: []domain.ReservedItem{{ProductID: "prod-1", ProductName: "Coffee", Quantity: 1, UnitPriceCents: 1299, SubtotalCents: 1299}},
		total:    1299,
	}
	cache := &fakeProductCache{}

	_, _, err := New(repo, cache).ReserveStock(ctx, "order-1", []domain.ReservationLine{{ProductID: "prod-1", Quantity: 1}})
	if err != nil {
		t.Fatalf("ReserveStock returned error: %v", err)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache invalidation after stock reserve, got %d delete calls", cache.deleteCalls)
	}
}

type fakeRepository struct {
	products  []domain.Product
	reserved  []domain.ReservedItem
	total     int64
	listCalls int
}

func (f *fakeRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	f.listCalls++
	return f.products, nil
}

func (f *fakeRepository) ReserveStock(ctx context.Context, orderID string, lines []domain.ReservationLine) ([]domain.ReservedItem, int64, error) {
	if orderID == "fail" {
		return nil, 0, errors.New("reserve failed")
	}
	return f.reserved, f.total, nil
}

func (f *fakeRepository) ReleaseReservation(ctx context.Context, orderID string) error {
	if orderID == "fail" {
		return errors.New("release failed")
	}
	return nil
}

func (f *fakeRepository) ConfirmReservation(ctx context.Context, orderID string) error {
	return nil
}

type fakeProductCache struct {
	products    []domain.Product
	hit         bool
	setCalls    int
	deleteCalls int
}

func (f *fakeProductCache) GetProducts(ctx context.Context) ([]domain.Product, bool, error) {
	return f.products, f.hit, nil
}

func (f *fakeProductCache) SetProducts(ctx context.Context, products []domain.Product) error {
	f.setCalls++
	f.products = products
	f.hit = true
	return nil
}

func (f *fakeProductCache) DeleteProducts(ctx context.Context) error {
	f.deleteCalls++
	f.products = nil
	f.hit = false
	return nil
}
