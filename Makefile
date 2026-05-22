.PHONY: help init-env setup boot all build build-api build-cli auth-build test test-fast test-amnesia test-api test-cli lint dev-up dev-down migrate migrate-down migrate-auth jet-gen swagger sdk-types dev-api dev-auth dev-app dev smoke clean

BIN := ./bin
DATABASE_URL ?= postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable

# ==========================================
# Onboarding & First-Time Setup (DevEx)
# ==========================================

help:
	@echo "Zenv-sh Makefile Commands:"
	@echo ""
	@echo "--- Setup ---"
	@echo "  make boot           - First-time setup (envs, dependencies, databases, migrations)"
	@echo "  make setup          - Install tools and code-gen dependencies"
	@echo ""
	@echo "--- Local Development ---"
	@echo "  make dev            - Start all services (API, Auth, App) concurrently"
	@echo "  make dev-api        - Start the Go API (Portless)"
	@echo "  make dev-auth       - Start the Auth service (Portless)"
	@echo "  make dev-app        - Start the Web App (Portless)"
	@echo "  make dev-up         - Start local databases (Postgres, Redis)"
	@echo "  make dev-down       - Stop local databases"
	@echo ""
	@echo "--- Code Generation & Migrations ---"
	@echo "  make migrate        - Apply Drizzle schema to the database"
	@echo "  make migrate-auth   - Run Auth DB migrations"
	@echo "  make jet-gen        - Generate Go-Jet types from schema"
	@echo "  make swagger        - Generate OpenAPI/Swagger docs"
	@echo "  make sdk-types      - Generate TypeScript types from OpenAPI"
	@echo ""
	@echo "--- Testing & CI ---"
	@echo "  make build          - Build API and CLI binaries"
	@echo "  make test           - Run all test suites"
	@echo "  make lint           - Run linters"
	@echo "  make smoke          - Run smoke tests"
	@echo "  make clean          - Remove built binaries"

init-env:
	@echo "Scaffolding .env files..."
	@test -f api/.env || cp api/.env.example api/.env
	@test -f apps/web/.env || cp apps/web/.env.example apps/web/.env
	@test -f apps/auth/.env || cp apps/auth/.env.example apps/auth/.env
	@test -f cli/.env || cp cli/.env.example cli/.env 2>/dev/null || true

setup: init-env
	@echo "Installing tools and dependencies..."
	pnpm install
	@echo "Ensuring code-gen tools are installed..."
	go install github.com/go-jet/jet/v2/cmd/jet@latest
	go install github.com/swaggo/swag/cmd/swag@latest

boot: setup dev-up
	@echo "Waiting for Postgres to accept connections..."
	@sleep 3
	@make migrate
	@make migrate-auth
	@echo "\n========================================"
	@echo "✅ Infrastructure and Database are ready!"
	@echo "========================================"
	@echo "To start developing, you can run:"
	@echo "  make dev"
	@echo "Or run these in separate terminal tabs:"
	@echo "  make dev-api"
	@echo "  make dev-auth"
	@echo "  make dev-app"

# ==========================================
# Build
# ==========================================

all: build

build: build-api build-cli

build-api:
	go build -o $(BIN)/zenv-api ./api/cmd/zenv-api

build-cli:
	go build -o $(BIN)/zenv ./cli/cmd/zenv

# ==========================================
# Auth server
# ==========================================

auth-build:
	pnpm -C apps/auth run build

migrate-auth:
	pnpm -C apps/auth run db:migrate

# ==========================================
# Test
# ==========================================

test:
	go test ./amnesia/... ./api/... ./cli/...

test-fast:
	go test ./amnesia/... ./cli/...

test-amnesia:
	go test -v -count=1 ./amnesia/...

test-api:
	go test -v -count=1 ./api/...

test-cli:
	go test -v -count=1 ./cli/...

# ==========================================
# Lint
# ==========================================

lint:
	golangci-lint run ./amnesia/... ./api/... ./cli/...

# ==========================================
# Dev infrastructure
# ==========================================

dev-up:
	docker compose up -d

dev-down:
	docker compose down -v

# ==========================================
# Migrations
# ==========================================

migrate:
	go run ./api/cmd/apply-drizzle

migrate-down:
	@echo "migrate-down is not supported for Drizzle-applied schema; restore from backup or reset the database." >&2
	@exit 1

# ==========================================
# Code Generation
# ==========================================

# --- Go-Jet codegen ---
jet-gen:
	~/go/bin/jet -dsn="$(DATABASE_URL)" -schema=public -path=./api/internal/store/gen

# --- OpenAPI / Swagger ---
swagger:
	~/go/bin/swag init -g api/cmd/zenv-api/main.go -o api/docs --parseDependency --parseInternal

# --- Generate TypeScript types from OpenAPI spec ---
sdk-types: swagger
	pnpm exec swagger2openapi api/docs/swagger.json -o api/docs/openapi.json
	pnpm -C packages/sdk exec openapi-typescript ../../api/docs/openapi.json -o src/api.d.ts
	cp packages/sdk/src/api.d.ts apps/web/src/lib/api.d.ts

# ==========================================
# Dev (Portless)
# ==========================================

dev-api:
	env $$(grep -v '^#' api/.env | xargs) portless api.zenv go run ./api/cmd/zenv-api

dev-auth:
	portless auth.zenv pnpm -C apps/auth run dev

dev-app:
	portless app.zenv pnpm -C apps/web dev

dev:
	@make -j 3 dev-api dev-auth dev-app

# ==========================================
# Smoke tests & Clean
# ==========================================

smoke: build
	./tests/smoke.sh

clean:
	rm -rf $(BIN)
