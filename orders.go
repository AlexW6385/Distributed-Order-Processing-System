package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateOrderRequest struct {
	CustomerEmail string                   `json:"customer_email"`
	Items         []CreateOrderItemRequest `json:"items"`
}

type CreateOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
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

type PayOrderRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
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

func (a *app) createOrder(c *gin.Context) {
	var request CreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	request.CustomerEmail = strings.TrimSpace(request.CustomerEmail)
	if request.CustomerEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_email is required"})
		return
	}

	if len(request.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one item is required"})
		return
	}

	for _, item := range request.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id is required"})
			return
		}
		if item.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be greater than zero"})
			return
		}
	}

	tx, err := a.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start order"})
		return
	}
	defer tx.Rollback()

	order := Order{
		CustomerEmail: request.CustomerEmail,
		Status:        "pending",
		Items:         make([]OrderItem, 0, len(request.Items)),
	}

	err = tx.QueryRowContext(c.Request.Context(), `
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	for _, requestItem := range request.Items {
		item, err := createOrderItem(c, tx, order.ID, requestItem)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusConflict, gin.H{"error": "product not found or insufficient stock"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order item"})
			return
		}

		order.TotalCents += item.SubtotalCents
		order.Items = append(order.Items, item)
	}

	err = tx.QueryRowContext(c.Request.Context(), `
		UPDATE orders
		SET total_cents = $1
		WHERE id = $2
		RETURNING updated_at
	`, order.TotalCents, order.ID).Scan(&order.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order total"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save order"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order": order})
}

func (a *app) getOrder(c *gin.Context) {
	order, err := a.findOrder(c, c.Param("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (a *app) payOrder(c *gin.Context) {
	orderID := strings.TrimSpace(c.Param("id"))

	var request PayOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.IdempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "idempotency_key is required"})
		return
	}

	tx, err := a.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start payment"})
		return
	}
	defer tx.Rollback()

	var order Order
	err = tx.QueryRowContext(c.Request.Context(), `
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
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	if order.Status == "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": "order is already paid"})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "order cannot be paid"})
		return
	}

	var payment Payment
	err = tx.QueryRowContext(c.Request.Context(), `
		INSERT INTO payments (order_id, idempotency_key, status, amount_cents)
		VALUES ($1, $2, 'succeeded', $3)
		RETURNING id, order_id, idempotency_key, status, amount_cents, created_at, updated_at
	`, order.ID, request.IdempotencyKey, order.TotalCents).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.IdempotencyKey,
		&payment.Status,
		&payment.AmountCents,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "payment already exists or idempotency key was used"})
		return
	}

	err = tx.QueryRowContext(c.Request.Context(), `
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
		RETURNING status, updated_at
	`, order.ID).Scan(&order.Status, &order.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save payment"})
		return
	}

	order, err = a.findOrder(c, order.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get paid order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order":   order,
		"payment": payment,
	})
}

func createOrderItem(c *gin.Context, tx *sql.Tx, orderID string, requestItem CreateOrderItemRequest) (OrderItem, error) {
	var item OrderItem
	item.OrderID = orderID
	item.ProductID = strings.TrimSpace(requestItem.ProductID)
	item.Quantity = requestItem.Quantity

	err := tx.QueryRowContext(c.Request.Context(), `
		UPDATE products
		SET stock_quantity = stock_quantity - $1
		WHERE id = $2 AND stock_quantity >= $1
		RETURNING price_cents
	`, item.Quantity, item.ProductID).Scan(&item.UnitPriceCents)
	if err != nil {
		return OrderItem{}, err
	}

	item.SubtotalCents = item.Quantity * item.UnitPriceCents

	err = tx.QueryRowContext(c.Request.Context(), `
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

func (a *app) findOrder(c *gin.Context, orderID string) (Order, error) {
	var order Order
	err := a.db.QueryRowContext(c.Request.Context(), `
		SELECT id, customer_email, status, total_cents, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, strings.TrimSpace(orderID)).Scan(
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

	rows, err := a.db.QueryContext(c.Request.Context(), `
		SELECT id, order_id, product_id, quantity, unit_price_cents, subtotal_cents, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, order.ID)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	order.Items = make([]OrderItem, 0)
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
			return Order{}, err
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return Order{}, err
	}

	return order, nil
}
