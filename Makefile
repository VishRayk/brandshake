SHELL := /bin/bash
export PATH := $(HOME)/.local/share/mise/shims:$(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: setup dev dev-api dev-ai dev-web db-start db-stop db-status db-check test test-api test-ai test-web lint build

setup: ## Install all dependencies
	cd go && go mod download
	cd python/ai && uv sync
	cd web && npm ci

dev: db-check ## Run API, AI service and web app together
	./scripts/dev.sh

db-start: ## Start the local PostgreSQL server
	sudo systemctl start postgresql

db-stop: ## Stop the local PostgreSQL server
	sudo systemctl stop postgresql

db-status:
	@systemctl is-active postgresql || true

db-check:
	@pg_isready -h localhost -p 5432 -q || { echo "PostgreSQL isn't running. Start it with: make db-start"; exit 1; }

dev-api:
	cd go && go run ./cmd/core-api

dev-ai:
	cd python/ai && uv run brandshake-ai

dev-web:
	cd web && npm run dev

test: test-api test-ai test-web ## Run every test suite

test-api: db-check
	cd go && go test -race ./...

test-ai:
	cd python/ai && uv run pytest -q

test-web:
	cd web && npm test

lint: ## Lint and type-check everything
	cd go && go vet ./... && test -z "$$(gofmt -l .)"
	cd python/ai && uv run ruff check . && uv run ruff format --check .
	cd web && npm run lint && npm run typecheck

build:
	cd go && go build -o bin/core-api ./cmd/core-api
	cd web && npm run build
