package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := appdb.Open(databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	runSchema(t, db)
	cleanTables(t, db)
	t.Cleanup(func() {
		cleanTables(t, db)
		db.Close()
	})

	return db
}

func InsertProduct(t *testing.T, db *sql.DB, sku string, priceCents int, stockQuantity int) string {
	t.Helper()

	var productID string
	err := db.QueryRow(`
		INSERT INTO products (sku, name, description, price_cents, stock_quantity)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, sku, sku, "Test product", priceCents, stockQuantity).Scan(&productID)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}

	return productID
}

func ProductStock(t *testing.T, db *sql.DB, productID string) int {
	t.Helper()

	var stock int
	if err := db.QueryRow(`SELECT stock_quantity FROM products WHERE id = $1`, productID).Scan(&stock); err != nil {
		t.Fatalf("select product stock: %v", err)
	}
	return stock
}

func CountRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

func runSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	schemaPath := filepath.Join("..", "..", "migrations", "init.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("run schema: %v", err)
	}
}

func cleanTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE outbox_events, payments, order_items, orders, products RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("clean tables: %v", err)
	}
}
