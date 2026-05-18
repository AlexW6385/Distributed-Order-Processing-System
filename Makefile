COMPOSE_DEV=docker compose
COMPOSE_PROD=docker compose -f docker-compose.prod.yml

.PHONY: proto tidy test up down reset logs ps prod-up prod-down

proto:
	./scripts/generate-proto.sh

tidy:
	go mod tidy

test:
	go test ./...

up:
	$(COMPOSE_DEV) up -d

down:
	$(COMPOSE_DEV) down

reset:
	$(COMPOSE_DEV) down -v

logs:
	$(COMPOSE_DEV) logs -f

ps:
	$(COMPOSE_DEV) ps

prod-up:
	$(COMPOSE_PROD) up -d --build

prod-down:
	$(COMPOSE_PROD) down
