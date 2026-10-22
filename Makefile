COMPOSE := docker compose -f infra/compose/docker-compose.yml
ifneq (,$(wildcard .env))
COMPOSE += --env-file .env
endif

.PHONY: up down lint test

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

lint:
	$(COMPOSE) config --quiet
	cd collector && test -z "$$(gofmt -l .)"
	cd collector && go vet ./...

test:
	./scripts/check_scaffold.sh
	cd collector && go test ./...
