# Distributed Order Processing System

A small order-processing system designed as separate services with a single HTTP gateway, internal gRPC communication, service-owned database tables, Redis idempotency and product caching, Kafka notifications, and an order outbox.

## Architecture

```text
Client
  |
  | HTTP JSON
  v
api-gateway
  | gRPC
  +--> product-service
  +--> order-service
          | gRPC
          +--> product-service
          +--> payment-service
          |
          | transactional outbox
          v
        Kafka topic: order.paid
          |
          v
notification-service
```

## Services

- `api-gateway`: public HTTP API. It translates JSON requests into gRPC calls and does not own business logic.
- `order-service`: owns order workflow, order tables, order items, and the outbox table.
- `product-service`: owns products, stock, stock reservations, and Redis-backed product list caching.
- `payment-service`: owns payments and Redis-backed idempotency keys.
- `notification-service`: consumes Kafka `order.paid` events and handles notification side effects.

Each business service is organized internally as:

```text
domain      business models
service     business rules and orchestration
repository  database access and SQL mapping
transport   gRPC or Kafka boundary code
migrations  tables owned by that service
```

## Run Locally

```bash
cp .env.example .env
make up
```

The API gateway listens on `http://localhost:8080`.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/products
```

Create an order:

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_email": "alex@example.com",
    "items": [
      {"product_id": "prod-coffee-001", "quantity": 2},
      {"product_id": "prod-mug-003", "quantity": 1}
    ]
  }'
```

Pay an order:

```bash
curl -X POST http://localhost:8080/orders/{order_id}/pay \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: checkout-123' \
  -d '{}'
```

## Design Notes

- The gateway is the only public service in Docker Compose.
- The services communicate internally through protobuf/gRPC.
- Product stock is reserved before an order becomes payable.
- Product listing uses Redis cache-aside reads and invalidates the cache when stock reservations change inventory.
- If payment fails, the order service releases the stock reservation and marks the order failed.
- When payment succeeds, the order service updates its own database and writes an `order.paid` outbox event in the same transaction.
- A background publisher sends pending outbox events to Kafka.
- Notification is asynchronous and does not block the checkout request.

## Useful Commands

```bash
make proto   # regenerate protobuf Go code
make test    # run Go tests
make up      # start local stack
make logs    # follow service logs
make reset   # stop and delete local volumes
```
