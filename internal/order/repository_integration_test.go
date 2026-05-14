package order

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/testutil"
)

func TestRepositoryCreateOrderPersistsOrderAndDecrementsStock(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-keyboard", 12999, 5)

	createdOrder, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if createdOrder.Status != "pending" {
		t.Fatalf("expected pending order, got %q", createdOrder.Status)
	}
	if createdOrder.TotalCents != 25998 {
		t.Fatalf("expected total 25998, got %d", createdOrder.TotalCents)
	}
	if len(createdOrder.Items) != 1 {
		t.Fatalf("expected 1 order item, got %d", len(createdOrder.Items))
	}
	if stock := testutil.ProductStock(t, db, productID); stock != 3 {
		t.Fatalf("expected stock 3, got %d", stock)
	}
}

func TestRepositoryCreateOrderRollsBackWhenStockIsInsufficient(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-mouse", 4999, 1)

	_, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 2},
		},
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}

	if stock := testutil.ProductStock(t, db, productID); stock != 1 {
		t.Fatalf("expected stock to remain 1, got %d", stock)
	}
	if orders := testutil.CountRows(t, db, "orders"); orders != 0 {
		t.Fatalf("expected no orders after rollback, got %d", orders)
	}
}

func TestRepositoryPayOrderCreatesPaymentAndMarksOrderPaid(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-monitor", 21999, 2)

	createdOrder, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	paidOrder, err := repository.Pay(context.Background(), createdOrder.ID, PayOrderRequest{
		IdempotencyKey: "payment-repo-monitor",
	})
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if paidOrder.Order.Status != "paid" {
		t.Fatalf("expected paid order, got %q", paidOrder.Order.Status)
	}
	if paidOrder.Payment.AmountCents != createdOrder.TotalCents {
		t.Fatalf("expected payment amount %d, got %d", createdOrder.TotalCents, paidOrder.Payment.AmountCents)
	}
}

func TestRepositoryPayOrderIsIdempotencyProtected(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-speakers", 7999, 2)

	createdOrder, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	_, err = repository.Pay(context.Background(), createdOrder.ID, PayOrderRequest{IdempotencyKey: "payment-speakers"})
	if err != nil {
		t.Fatalf("first payment: %v", err)
	}

	_, err = repository.Pay(context.Background(), createdOrder.ID, PayOrderRequest{IdempotencyKey: "payment-speakers"})
	if !errors.Is(err, ErrAlreadyPaid) {
		t.Fatalf("expected ErrAlreadyPaid, got %v", err)
	}
}

func TestRepositoryCreateOrderHandlesConcurrentStockDecrement(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-limited-stock", 1000, 1)

	const workers = 10
	results := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repository.Create(context.Background(), CreateOrderRequest{
				CustomerEmail: fmt.Sprintf("customer-%d@example.com", i),
				Items: []CreateOrderItemRequest{
					{ProductID: productID, Quantity: 1},
				},
			})
			results <- err
		}(i)
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
		if errors.Is(err, ErrInsufficientStock) {
			stockFailures++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful order, got %d", successes)
	}
	if stockFailures != workers-1 {
		t.Fatalf("expected %d stock failures, got %d", workers-1, stockFailures)
	}
	if stock := testutil.ProductStock(t, db, productID); stock != 0 {
		t.Fatalf("expected stock 0, got %d", stock)
	}
}
