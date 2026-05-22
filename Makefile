.PHONY: help init-env setup boot dev-infra db-push db-studio dev-api dev-dashboard dev-docs dev-cli down clean build-api build-cli build-dashboard build-docs build test lint format docker-prod

# ==========================================
# Onboarding & First-Time Setup (DevEx)
# ==========================================

help:
	@echo "Zenv-sh Makefile Commands:"
	@echo ""
	@echo "--- Development ---"
	@echo "  make boot           - First-time setup (envs, dependencies, databases)"
	@echo "  make dev-api        - Start the Go API with hot-reloading"
	@echo "  make dev-dashboard  - Start the Web Dashboard"
	@echo "  make dev-docs       - Start the Documentation Site"
	@echo "  make dev-cli        - Run the CLI locally"
	@echo "  make dev-infra      - Start local databases (Postgres, Redis)"
	@echo "  make db-push        - Push Drizzle schema to the database"
	@echo "  make db-studio      - Open Drizzle Studio to inspect the database"
	@echo ""
	@echo "--- Operations & CI ---"
	@echo "  make build          - Build all production binaries and apps"
	@echo "  make test           - Run all test suites"
	@echo "  make lint           - Run linters across Go and TypeScript"
	@echo "  make down           - Stop all Docker containers"
	@echo "  make clean          - Clean build artifacts and node_modules"

init-env:
	@echo "Scaffolding .env files..."
	@test -f api/.env || cp api/.env.example api/.env
	@test -f apps/web/.env || cp apps/web/.env.example apps/web/.env
	@test -f apps/auth/.env || cp apps/auth/.env.example apps/auth/.env
	@test -f cli/.env || cp cli/.env.example cli/.env 2>/dev/null || true

setup:
	@echo "Installing tools and dependencies..."
	go install github.com/cosmtrek/air@latest
	pnpm install

boot: init-env setup dev-infra
	@echo "Waiting for Postgres to accept connections..."
	@sleep 3
	@make db-push
	@echo "\n========================================"
	@echo "✅ Infrastructure and Database are ready!"
	@echo "========================================"
	@echo "To start developing, open your required terminal tabs. For example:"
	@echo "  Tab 1: make dev-api"
	@echo "  Tab 2: make dev-dashboard"

# ==========================================
# Granular Local Development
# ==========================================

dev-infra:
	@echo "Starting Postgres and Redis..."
	docker compose -f docker-compose.yml up -d

db-push:
	@echo "Running database migrations..."
	pnpm --filter ./apps/auth run db:push

db-studio:
	@echo "Starting Drizzle Studio..."
	pnpm --filter ./apps/auth run db:studio

dev-api:
	@echo "Starting Go API..."
	cd api && go run github.com/cosmtrek/air@latest -c .air.toml

dev-dashboard:
	@echo "Starting Web Dashboard..."
	pnpm --filter ./apps/web run dev

dev-docs:
	@echo "Starting Documentation Site..."
	pnpm --filter ./apps/docs run dev

dev-cli:
	@echo "Starting CLI development mode..."
	cd cli && go run cmd/zenv/main.go

down:
	@echo "Stopping infrastructure..."
	docker compose -f docker-compose.yml down
	docker compose -f docker-compose.prod.yml down 2>/dev/null || true

# ==========================================
# Building & Compilation
# ==========================================

build-api:
	@echo "Building Go API..."
	cd api && go build -o ../bin/zenv-api cmd/zenv-api/main.go

build-cli:
	@echo "Building CLI..."
	cd cli && go build -o ../bin/zenv cmd/zenv/main.go

build-dashboard:
	@echo "Building web dashboard..."
	pnpm --filter ./apps/web run build

build-docs:
	@echo "Building docs..."
	pnpm --filter ./apps/docs run build

build: build-api build-cli build-dashboard build-docs
	@echo "All builds completed."

# ==========================================
# Testing & Code Quality
# ==========================================

test:
	@echo "Running test suites..."
	cd api && go test -v ./...
	cd packages/amnesia && pnpm test
	pnpm --filter ./apps/web run test

lint:
	@echo "Running linters..."
	cd api && golangci-lint run
	pnpm --parallel -r run lint

format:
	@echo "Formatting code..."
	cd api && go fmt ./...
	pnpm --parallel -r run format

# ==========================================
# Production & Docker
# ==========================================

docker-prod:
	@echo "Building and starting production containers..."
	docker compose -f docker-compose.prod.yml up -d --build

clean: down
	@echo "Cleaning up environment..."
	rm -rf bin/
	rm -rf node_modules apps/web/node_modules apps/auth/node_modules apps/docs/node_modules packages/amnesia/node_modules packages/sdk/node_modules
	pnpm store prune
