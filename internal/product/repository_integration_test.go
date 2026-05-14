package product

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/testutil"
)

func TestRepositoryReserveStockDecrementsStock(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "product-reserve", 12999, 5)

	reservation, err := repository.ReserveStock(context.Background(), productID, 2)
	if err != nil {
		t.Fatalf("reserve stock: %v", err)
	}

	if reservation.SubtotalCents != 25998 {
		t.Fatalf("expected subtotal 25998, got %d", reservation.SubtotalCents)
	}
	if stock := testutil.ProductStock(t, db, productID); stock != 3 {
		t.Fatalf("expected stock 3, got %d", stock)
	}
}

func TestRepositoryReserveStockFailsWhenStockIsInsufficient(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "product-insufficient", 4999, 1)

	_, err := repository.ReserveStock(context.Background(), productID, 2)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if stock := testutil.ProductStock(t, db, productID); stock != 1 {
		t.Fatalf("expected stock to remain 1, got %d", stock)
	}
}

func TestRepositoryReserveStockHandlesConcurrentDecrement(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "product-concurrent", 1000, 1)

	const workers = 10
	results := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repository.ReserveStock(context.Background(), productID, 1)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	successes := 0
	stockFailures := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, sql.ErrNoRows) {
			stockFailures++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful reservation, got %d", successes)
	}
	if stockFailures != workers-1 {
		t.Fatalf("expected %d stock failures, got %d", workers-1, stockFailures)
	}
	if stock := testutil.ProductStock(t, db, productID); stock != 0 {
		t.Fatalf("expected stock 0, got %d", stock)
	}
}
