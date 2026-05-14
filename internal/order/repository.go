package order

import (
	"context"
	"database/sql"
	"errors"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, request CreateOrderRequest) (Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	order := Order{
		CustomerEmail: request.CustomerEmail,
		Status:        "pending",
		Items:         make([]OrderItem, 0, len(request.Items)),
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO orders (customer_email, status)
		VALUES ($1, $2)
		RETURNING id, total_cents, created_at, updated_at
	`, order.CustomerEmail, order.Status).Scan(
		&order.ID,
		&order.TotalCents,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return Order{}, err
	}

	for _, requestItem := range request.Items {
		item, err := createOrderItem(ctx, tx, order.ID, requestItem)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Order{}, ErrInsufficientStock
			}
			return Order{}, err
		}

		order.TotalCents += item.SubtotalCents
		order.Items = append(order.Items, item)
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE orders
		SET total_cents = $1
		WHERE id = $2
		RETURNING updated_at
	`, order.TotalCents, order.ID).Scan(&order.UpdatedAt)
	if err != nil {
		return Order{}, err
	}

	if err := tx.Commit(); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (r *Repository) Find(ctx context.Context, orderID string) (Order, error) {
	var order Order
	err := r.db.QueryRowContext(ctx, `
		SELECT id, customer_email, status, total_cents, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, orderID).Scan(
		&order.ID,
		&order.CustomerEmail,
		&order.Status,
		&order.TotalCents,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return Order{}, err
	}

	items, err := r.findItems(ctx, order.ID)
	if err != nil {
		return Order{}, err
	}
	order.Items = items

	return order, nil
}

func (r *Repository) Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return PaidOrder{}, err
	}
	defer tx.Rollback()

	var paidOrder PaidOrder
	err = tx.QueryRowContext(ctx, `
		SELECT id, customer_email, status, total_cents, created_at, updated_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, orderID).Scan(
		&paidOrder.Order.ID,
		&paidOrder.Order.CustomerEmail,
		&paidOrder.Order.Status,
		&paidOrder.Order.TotalCents,
		&paidOrder.Order.CreatedAt,
		&paidOrder.Order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaidOrder{}, ErrNotFound
		}
		return PaidOrder{}, err
	}

	if paidOrder.Order.Status == "paid" {
		return PaidOrder{}, ErrAlreadyPaid
	}

	if paidOrder.Order.Status != "pending" {
		return PaidOrder{}, ErrCannotBePaid
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO payments (order_id, idempotency_key, status, amount_cents)
		VALUES ($1, $2, 'succeeded', $3)
		RETURNING id, order_id, idempotency_key, status, amount_cents, created_at, updated_at
	`, paidOrder.Order.ID, request.IdempotencyKey, paidOrder.Order.TotalCents).Scan(
		&paidOrder.Payment.ID,
		&paidOrder.Payment.OrderID,
		&paidOrder.Payment.IdempotencyKey,
		&paidOrder.Payment.Status,
		&paidOrder.Payment.AmountCents,
		&paidOrder.Payment.CreatedAt,
		&paidOrder.Payment.UpdatedAt,
	)
	if err != nil {
		return PaidOrder{}, ErrConflict
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
		RETURNING status, updated_at
	`, paidOrder.Order.ID).Scan(&paidOrder.Order.Status, &paidOrder.Order.UpdatedAt)
	if err != nil {
		return PaidOrder{}, err
	}

	if err := tx.Commit(); err != nil {
		return PaidOrder{}, err
	}

	items, err := r.findItems(ctx, paidOrder.Order.ID)
	if err != nil {
		return PaidOrder{}, err
	}
	paidOrder.Order.Items = items

	return paidOrder, nil
}

func createOrderItem(ctx context.Context, tx *sql.Tx, orderID string, requestItem CreateOrderItemRequest) (OrderItem, error) {
	item := OrderItem{
		OrderID:   orderID,
		ProductID: requestItem.ProductID,
		Quantity:  requestItem.Quantity,
	}

	err := tx.QueryRowContext(ctx, `
		UPDATE products
		SET stock_quantity = stock_quantity - $1
		WHERE id = $2 AND stock_quantity >= $1
		RETURNING price_cents
	`, item.Quantity, item.ProductID).Scan(&item.UnitPriceCents)
	if err != nil {
		return OrderItem{}, err
	}

	item.SubtotalCents = item.Quantity * item.UnitPriceCents

	err = tx.QueryRowContext(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, subtotal_cents)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, item.OrderID, item.ProductID, item.Quantity, item.UnitPriceCents, item.SubtotalCents).Scan(
		&item.ID,
		&item.CreatedAt,
	)
	if err != nil {
		return OrderItem{}, err
	}

	return item, nil
}

func (r *Repository) findItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, quantity, unit_price_cents, subtotal_cents, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.SubtotalCents,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
