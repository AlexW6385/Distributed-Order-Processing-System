COMPOSE_DEV=docker compose
COMPOSE_PROD=docker compose -f docker-compose.prod.yml
TEST_DATABASE_URL?=postgres://postgres:postgres@localhost:5432/distributed_order_processing_system_test?sslmode=disable
TEST_REDIS_ADDR?=localhost:6379

.PHONY: up down reset logs ps test test-integration proto prod-up prod-down prod-logs prod-ps prod-reset

up:
	$(COMPOSE_DEV) up -d --build

down:
	$(COMPOSE_DEV) down

reset:
	$(COMPOSE_DEV) down -v
	$(COMPOSE_DEV) up -d --build

logs:
	$(COMPOSE_DEV) logs -f

ps:
	$(COMPOSE_DEV) ps

test:
	GOWORK=off go test ./...

test-integration:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' TEST_REDIS_ADDR='$(TEST_REDIS_ADDR)' GOWORK=off go test -p 1 ./...

proto:
	./scripts/generate-proto.sh

prod-up:
	$(COMPOSE_PROD) up -d --build

prod-down:
	$(COMPOSE_PROD) down

prod-reset:
	$(COMPOSE_PROD) down -v
	$(COMPOSE_PROD) up -d --build

prod-logs:
	$(COMPOSE_PROD) logs -f

prod-ps:
	$(COMPOSE_PROD) ps
