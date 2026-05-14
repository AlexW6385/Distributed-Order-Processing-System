package order

import (
	"context"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/testutil"
)

func TestRepositoryCreateOrderPersistsOrderAndItems(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-keyboard", 12999, 5)

	createdOrder, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 2},
		},
	}, []ReservedStock{
		{ProductID: productID, Quantity: 2, UnitPriceCents: 12999, SubtotalCents: 25998},
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
}

func TestRepositoryMarkPaidMarksOrderPaid(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	productID := testutil.InsertProduct(t, db, "repo-monitor", 21999, 2)

	createdOrder, err := repository.Create(context.Background(), CreateOrderRequest{
		CustomerEmail: "alex@example.com",
		Items: []CreateOrderItemRequest{
			{ProductID: productID, Quantity: 1},
		},
	}, []ReservedStock{
		{ProductID: productID, Quantity: 1, UnitPriceCents: 21999, SubtotalCents: 21999},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	paidOrder, err := repository.MarkPaid(context.Background(), createdOrder.ID)
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}

	if paidOrder.Status != "paid" {
		t.Fatalf("expected paid order, got %q", paidOrder.Status)
	}
}
