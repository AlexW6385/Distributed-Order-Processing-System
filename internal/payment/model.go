package payment

import "time"

type PayOrderRequest struct {
	OrderID        string
	IdempotencyKey string
	AmountCents    int
}

type Payment struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Status         string    `json:"status"`
	AmountCents    int       `json:"amount_cents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
