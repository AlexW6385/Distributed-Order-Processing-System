package product

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

func (r *Repository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sku, name, description, price_cents, stock_quantity, created_at, updated_at
		FROM products
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.PriceCents,
			&product.StockQuantity,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *Repository) ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error) {
	var reservation ReservedStock
	reservation.ProductID = productID
	reservation.Quantity = quantity

	err := r.db.QueryRowContext(ctx, `
		UPDATE products
		SET stock_quantity = stock_quantity - $1
		WHERE id = $2 AND stock_quantity >= $1
		RETURNING price_cents
	`, quantity, productID).Scan(&reservation.UnitPriceCents)
	if err != nil {
		return ReservedStock{}, err
	}

	reservation.SubtotalCents = reservation.UnitPriceCents * reservation.Quantity
	return reservation, nil
}
