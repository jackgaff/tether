COMPOSE := docker compose
.DEFAULT_GOAL := help

.PHONY: help init install check start stop status up down restart rebuild build logs ps db-reset clean api-logs web-logs db-logs prompt-test prompt-test-logs

help:
	@printf "Available targets:\n"
	@printf "  make init      Create .env from .env.example if needed and install deps\n"
	@printf "  make install   Install Bun workspace dependencies with the lockfile\n"
	@printf "  make check     Run the repo verification suite\n"
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

db-reset:
	$(COMPOSE) down
	docker volume rm nova-echoes_postgres-data || true
	$(COMPOSE) up -d

clean:
	$(COMPOSE) down -v
