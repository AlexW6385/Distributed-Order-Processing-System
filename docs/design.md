# Design Summary

This project uses a gateway plus domain services.

The HTTP API is defined in `openapi.yaml`. External users send JSON to the gateway. The gateway converts the request into protobuf messages and calls internal gRPC services.

The core business model belongs to each service's `domain` package. Transport code converts JSON or protobuf into domain values. Repository code converts database rows into domain values. The service layer only works with domain values.

Postgres is one local database container, but tables are owned by services:

- product-service owns `products` and `stock_reservations`
- order-service owns `orders`, `order_items`, and `outbox_events`
- payment-service owns `payments`

Other services should treat a service-owned table as private, even though local Docker Compose uses one Postgres instance for convenience.

Kafka is shared infrastructure. The order service writes important events to its own outbox table first, then a publisher sends them to Kafka. This avoids losing the notification event after the order has already been marked paid.
