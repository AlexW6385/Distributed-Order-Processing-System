package order

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/events"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, request CreateOrderRequest, reservedItems []ReservedStock) (Order, error) {
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

	for _, reservedItem := range reservedItems {
		item, err := createOrderItem(ctx, tx, order.ID, reservedItem)
		if err != nil {
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

func (r *Repository) MarkPaid(ctx context.Context, orderID string, event events.OrderPaidEvent) (Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	var order Order
	err = tx.QueryRowContext(ctx, `
		SELECT id, customer_email, status, total_cents, created_at, updated_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
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

	err = tx.QueryRowContext(ctx, `
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
		RETURNING status, updated_at
	`, order.ID).Scan(&order.Status, &order.UpdatedAt)
	if err != nil {
		return Order{}, err
	}

	if err := createOutboxEvent(ctx, tx, order.ID, event); err != nil {
		return Order{}, err
	}

	if err := tx.Commit(); err != nil {
		return Order{}, err
	}

	items, err := r.findItems(ctx, order.ID)
	if err != nil {
		return Order{}, err
	}
	order.Items = items

	return order, nil
}

func createOrderItem(ctx context.Context, tx *sql.Tx, orderID string, reservedItem ReservedStock) (OrderItem, error) {
	item := OrderItem{
		OrderID:        orderID,
		ProductID:      reservedItem.ProductID,
		Quantity:       reservedItem.Quantity,
		UnitPriceCents: reservedItem.UnitPriceCents,
		SubtotalCents:  reservedItem.SubtotalCents,
	}

	err := tx.QueryRowContext(ctx, `
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

func createOutboxEvent(ctx context.Context, tx *sql.Tx, orderID string, event events.OrderPaidEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ('order', $1, $2, $3)
	`, orderID, events.OrderPaidEventType, payload)
	return err
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
