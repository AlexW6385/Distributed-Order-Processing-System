package domain

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Stock       int32
}

type ReservationLine struct {
	ProductID string
	Quantity  int32
}

type ReservedItem struct {
	ProductID      string
	ProductName    string
	Quantity       int32
	UnitPriceCents int64
	SubtotalCents  int64
}
