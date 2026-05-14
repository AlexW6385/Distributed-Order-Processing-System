# Distributed Order Processing System

A Go order-processing MVP split into small services. The public API is served by `order-service`, and it talks to `product-service` and `payment-service` over gRPC.

## Current Scope

The app currently supports:

- Listing products
- Creating orders
- Reading orders by ID
- Simulated order payment
- Basic health check
- Product gRPC service
- Payment gRPC service
- Order HTTP service that calls product and payment services over gRPC

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
├── docker-compose.yml
├── migrations/
│   └── init.sql
├── proto/
│   ├── product/v1/product.proto
│   └── payment/v1/payment.proto
├── gen/
│   ├── product/v1/
│   └── payment/v1/
├── services/
│   ├── order-service/
│   ├── product-service/
│   └── payment-service/
└── internal/
    ├── config/
    ├── db/
    ├── health/
    ├── product/
    │   ├── handler.go
    │   ├── model.go
    │   ├── repository.go
    │   └── service.go
    ├── payment/
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

The `product`, `payment`, and `order` packages are still organized as vertical modules inside `internal/`. Each service entrypoint wires only the module code it owns or calls over gRPC.

## Requirements

- Go
- Docker
- Docker Compose

## Run Locally

Start all services:

```bash
docker compose up -d
```

This starts:

- Postgres
- Redis
- `product-service` on gRPC port `50051`
- `payment-service` on gRPC port `50052`
- `order-service` on HTTP port `8080`

Run a single service locally:

```bash
go run ./services/order-service
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
DATABASE_URL='postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable' REDIS_ADDR='localhost:6379' PRODUCT_SERVICE_ADDR='localhost:50051' PAYMENT_SERVICE_ADDR='localhost:50052' PORT=8080 go run ./services/order-service
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

The full integration suite uses one shared test database, so run it serially when all integration tests are enabled:

```bash
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/distributed_order_processing_system_test?sslmode=disable' TEST_REDIS_ADDR='localhost:6379' go test -p 1 ./...
```

The integration test helper applies `migrations/init.sql` and truncates test tables before and after each test run.

## CI

GitHub Actions runs on pushes and pull requests to `main`.

CI checks:

- Go formatting
- Unit tests
- Database integration tests using a temporary Postgres service
- Redis integration tests using a temporary Redis service

## Protobuf

gRPC definitions live in:

```text
proto/product/v1/product.proto
proto/payment/v1/payment.proto
```

Generated Go code is committed under:

```text
gen/product/v1
gen/payment/v1
```

Regenerate protobuf code:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
./scripts/generate-proto.sh
```
