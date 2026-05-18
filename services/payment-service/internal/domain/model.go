package domain

type Payment struct {
	ID             string
	OrderID        string
	IdempotencyKey string
	AmountCents    int64
	Status         string
}
