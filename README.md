# Distributed Order Processing System

A Go order-processing MVP built as a modular monolith first, with the code organized so each business module can later be split into its own microservice.

## Current Scope

The app currently supports:

- Listing products
- Creating orders
- Reading orders by ID
- Simulated order payment
- Basic health check

HTTP endpoints:

```text
GET  /health
GET  /products
POST /orders
GET  /orders/:id
POST /orders/:id/pay
```

## Project Structure

```text
.
├── main.go
├── docker-compose.yml
├── migrations/
│   └── init.sql
└── internal/
    ├── config/
    ├── db/
    ├── health/
    ├── product/
    │   ├── handler.go
    │   ├── model.go
    │   ├── repository.go
    │   └── service.go
    └── order/
        ├── errors.go
        ├── handler.go
        ├── model.go
        ├── repository.go
        └── service.go
```

`product` and `order` are organized as vertical modules. Each module owns its HTTP handler, service logic, data model, and repository code.

## Requirements

- Go
- Docker
- Docker Compose

## Run Locally

Start Postgres:

```bash
docker compose up -d
```

Run the app:

```bash
go run .
```

The default database URL is:

```text
postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable
```

The default Redis address is:

```text
localhost:6379
```

You can override it:

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable' REDIS_ADDR='localhost:6379' PORT=8080 go run .
```

## Example Requests

List products:

```bash
curl http://localhost:8080/products
```

Create an order:

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_email": "alex@example.com",
    "items": [
      {
        "product_id": "PRODUCT_ID",
        "quantity": 1
      }
    ]
  }'
```

Get an order:

```bash
curl http://localhost:8080/orders/ORDER_ID
```

Pay an order:

```bash
curl -X POST http://localhost:8080/orders/ORDER_ID/pay \
  -H 'Content-Type: application/json' \
  -d '{
    "idempotency_key": "payment-001"
  }'
```

## Tests

Run unit tests:

```bash
go test ./...
```

Database integration tests run only when `TEST_DATABASE_URL` is set. Redis integration tests run only when `TEST_REDIS_ADDR` is set. This prevents accidental writes to the development database or local Redis instance.

Create a local test database:

```bash
docker exec distributed-order-processing-system-postgres createdb -U postgres distributed_order_processing_system_test
```

Run the full test suite:

```bash
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/distributed_order_processing_system_test?sslmode=disable' TEST_REDIS_ADDR='localhost:6379' go test ./...
```

The integration test helper applies `migrations/init.sql` and truncates test tables before and after each test run.

## CI

GitHub Actions runs on pushes and pull requests to `main`.

CI checks:

- Go formatting
- Unit tests
- Database integration tests using a temporary Postgres service
- Redis integration tests using a temporary Redis service
