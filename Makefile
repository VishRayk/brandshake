SHELL := /bin/bash
export PATH := $(HOME)/.local/share/mise/shims:$(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: setup dev dev-api dev-ai dev-web test test-api test-ai test-web lint build

setup: ## Install all dependencies
	cd go && go mod download
	cd python/ai && uv sync
	cd web && npm ci

dev: ## Run API, AI service and web app together
	./scripts/dev.sh

dev-api:
	cd go && go run ./cmd/core-api

dev-ai:
	cd python/ai && uv run brandshake-ai

dev-web:
	cd web && npm run dev

test: test-api test-ai test-web ## Run every test suite

test-api:
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
