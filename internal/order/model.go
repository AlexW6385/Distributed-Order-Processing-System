package order

import "time"

type CreateOrderRequest struct {
	CustomerEmail string
	Items         []CreateOrderItemRequest
}

type CreateOrderItemRequest struct {
	ProductID string
	Quantity  int
}

type PayOrderRequest struct {
	IdempotencyKey string
}

type Order struct {
	ID            string      `json:"id"`
	CustomerEmail string      `json:"customer_email"`
	Status        string      `json:"status"`
	TotalCents    int         `json:"total_cents"`
	Items         []OrderItem `json:"items"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	UnitPriceCents int       `json:"unit_price_cents"`
	SubtotalCents  int       `json:"subtotal_cents"`
	CreatedAt      time.Time `json:"created_at"`
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

type PaidOrder struct {
	Order   Order   `json:"order"`
	Payment Payment `json:"payment"`
}
