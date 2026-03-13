COMPOSE ?= docker compose
.DEFAULT_GOAL := help

.PHONY: help init install check check-api check-web check-test test-api-integration compose-config start stop status up down restart rebuild build logs ps db-reset clean api-logs web-logs db-logs prompt-test prompt-test-logs prompt-test-down

help:
	@printf "Available targets:\n"
	@printf "  make init      Create .env from .env.example if needed and install deps\n"
	@printf "  make install   Install Bun workspace dependencies with the lockfile\n"
	@printf "  make check     Run the repo verification suite\n"
	@printf "  make check-api Run Go tests, race detector, and go vet\n"
	@printf "  make check-web Typecheck and build the main web app\n"
	@printf "  make check-test Typecheck and build the prompt lab app\n"
	@printf "  make test-api-integration Run Postgres-backed API integration tests\n"
	@printf "  make compose-config Validate docker compose configuration\n"
	@printf "  make start     Start db, api, and web in the background\n"
	@printf "  make stop      Stop the full stack\n"
	@printf "  make status    Show service status\n"
	@printf "  make up        Start db, api, and web in the background\n"
	@printf "  make down      Stop the full stack\n"
	@printf "  make restart   Recreate the full stack\n"
	@printf "  make rebuild   Rebuild images and start the full stack\n"
	@printf "  make build     Rebuild the api and web images\n"
	@printf "  make logs      Tail logs for all services\n"
	@printf "  make ps        Show service status\n"
	@printf "  make prompt-test      Start the standalone prompt lab service\n"
	@printf "  make prompt-test-logs Tail logs for the prompt lab service\n"
	@printf "  make db-reset  Remove Postgres data and restart the full stack\n"
	@printf "  make clean     Stop the stack and remove all compose volumes\n"

init:
	@if [ ! -f .env ]; then cp .env.example .env; fi
	$(MAKE) install

install:
	bun install --frozen-lockfile

check:
	bun run check

check-api:
	bun run check:api

check-web:
	bun run check:web

check-test:
	bun run check:test

test-api-integration:
	TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5433/nova_echoes?sslmode=disable} bun run test:api:integration

compose-config:
	$(COMPOSE) config > /dev/null

start: up

stop: down

status: ps

up:
	$(COMPOSE) up -d db api web

down:
	$(COMPOSE) down

restart: down up

rebuild:
	$(COMPOSE) up -d --build db api web

build:
	$(COMPOSE) build api web

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

api-logs:
	$(COMPOSE) logs -f api

web-logs:
	$(COMPOSE) logs -f web

db-logs:
	$(COMPOSE) logs -f db

prompt-test:
	$(COMPOSE) up -d --build test

prompt-test-logs:
	$(COMPOSE) logs -f test

prompt-test-down:
	$(COMPOSE) stop test

db-reset:
	$(COMPOSE) down
	docker volume rm nova-echoes_postgres-data || true
	$(COMPOSE) up -d

clean:
	$(COMPOSE) down -v
