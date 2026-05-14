package product

import (
	"context"
	"testing"
)

type fakeStore struct {
	products []Product
	err      error
	calls    int
}

func (s *fakeStore) List(ctx context.Context) ([]Product, error) {
	s.calls++
	return s.products, s.err
}

func (s *fakeStore) ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error) {
	return ReservedStock{
		ProductID:      productID,
		Quantity:       quantity,
		UnitPriceCents: 1000,
		SubtotalCents:  quantity * 1000,
	}, s.err
}

type fakeCache struct {
	products    []Product
	hit         bool
	getErr      error
	setProducts []Product
	setCalls    int
}

func (c *fakeCache) GetProducts(ctx context.Context) ([]Product, bool, error) {
	return c.products, c.hit, c.getErr
}

func (c *fakeCache) SetProducts(ctx context.Context, products []Product) error {
	c.setCalls++
	c.setProducts = products
	return nil
}

func (c *fakeCache) DeleteProducts(ctx context.Context) error {
	return nil
}

func TestServiceListReturnsProducts(t *testing.T) {
	service := NewService(&fakeStore{
		products: []Product{
			{ID: "product-1", SKU: "keyboard-001", Name: "Keyboard"},
			{ID: "product-2", SKU: "mouse-001", Name: "Mouse"},
		},
	})

	products, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list products: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].SKU != "keyboard-001" {
		t.Fatalf("expected first SKU keyboard-001, got %q", products[0].SKU)
	}
}

func TestServiceListReturnsCachedProducts(t *testing.T) {
	store := &fakeStore{
		products: []Product{{ID: "database-product", SKU: "db-001"}},
	}
	cache := &fakeCache{
		hit:      true,
		products: []Product{{ID: "cached-product", SKU: "cache-001"}},
	}
	service := NewService(store, cache)

	products, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list products: %v", err)
	}

	if len(products) != 1 || products[0].ID != "cached-product" {
		t.Fatalf("expected cached product, got %+v", products)
	}
	if store.calls != 0 {
		t.Fatalf("expected store not to be called, got %d calls", store.calls)
	}
}

func TestServiceListCachesRepositoryProductsOnMiss(t *testing.T) {
	store := &fakeStore{
		products: []Product{{ID: "database-product", SKU: "db-001"}},
	}
	cache := &fakeCache{}
	service := NewService(store, cache)

	products, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list products: %v", err)
	}

	if len(products) != 1 || products[0].ID != "database-product" {
		t.Fatalf("expected database product, got %+v", products)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache set once, got %d", cache.setCalls)
	}
	if len(cache.setProducts) != 1 || cache.setProducts[0].ID != "database-product" {
		t.Fatalf("expected cache to store database product, got %+v", cache.setProducts)
	}
}
