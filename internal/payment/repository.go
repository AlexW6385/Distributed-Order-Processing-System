package payment

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, request PayOrderRequest) (Payment, error) {
	var payment Payment
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payments (order_id, idempotency_key, status, amount_cents)
		VALUES ($1, $2, 'succeeded', $3)
		RETURNING id, order_id, idempotency_key, status, amount_cents, created_at, updated_at
	`, request.OrderID, request.IdempotencyKey, request.AmountCents).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.IdempotencyKey,
		&payment.Status,
		&payment.AmountCents,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return Payment{}, ErrConflict
	}

	return payment, nil
}
