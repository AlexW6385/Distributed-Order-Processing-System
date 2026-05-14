package payment

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/testutil"
)

func TestRepositoryCreatePaymentPersistsPayment(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	orderID := insertOrder(t, db, "payment-repo-order-1")

	createdPayment, err := repository.Create(context.Background(), PayOrderRequest{
		OrderID:        orderID,
		IdempotencyKey: "payment-repo-1",
		AmountCents:    12999,
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if createdPayment.Status != "succeeded" {
		t.Fatalf("expected succeeded payment, got %q", createdPayment.Status)
	}
	if createdPayment.AmountCents != 12999 {
		t.Fatalf("expected amount 12999, got %d", createdPayment.AmountCents)
	}
}

func TestRepositoryCreatePaymentRejectsDuplicateIdempotencyKey(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repository := NewRepository(db)
	orderID := insertOrder(t, db, "payment-repo-order-duplicate")
	request := PayOrderRequest{
		OrderID:        orderID,
		IdempotencyKey: "payment-repo-duplicate",
		AmountCents:    12999,
	}

	if _, err := repository.Create(context.Background(), request); err != nil {
		t.Fatalf("first payment: %v", err)
	}

	_, err := repository.Create(context.Background(), request)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func insertOrder(t *testing.T, db interface {
	QueryRow(query string, args ...any) *sql.Row
}, email string) string {
	t.Helper()

	var orderID string
	err := db.QueryRow(`
		INSERT INTO orders (customer_email, status, total_cents)
		VALUES ($1, 'pending', 12999)
		RETURNING id
	`, email).Scan(&orderID)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	return orderID
}
