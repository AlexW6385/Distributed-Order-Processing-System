package product

import (
	"context"
	"testing"
)

type fakeStore struct {
	products []Product
	err      error
}

func (s *fakeStore) List(ctx context.Context) ([]Product, error) {
	return s.products, s.err
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
