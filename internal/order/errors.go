package order

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrAlreadyPaid       = errors.New("order is already paid")
	ErrCannotBePaid      = errors.New("order cannot be paid")
	ErrInsufficientStock = errors.New("product not found or insufficient stock")
)
