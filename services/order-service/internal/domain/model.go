package domain

import "time"

const (
	StatusPending       = "pending"
	StatusStockReserved = "stock_reserved"
	StatusPaid          = "paid"
	StatusFailed        = "failed"
)

type Order struct {
	ID            string
	CustomerEmail string
	Status        string
	TotalCents    int64
	FailureReason string
	Items         []Item
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Item struct {
	ProductID      string
	ProductName    string
	Quantity       int32
	UnitPriceCents int64
	SubtotalCents  int64
}

type ItemInput struct {
	ProductID string
	Quantity  int32
}

type OutboxEvent struct {
	ID        string
	EventType string
	Payload   []byte
}
