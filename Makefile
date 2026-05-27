.PHONY: help init-env setup boot all build build-api build-cli auth-build test test-fast test-amnesia test-api test-cli lint dev-build dev-up dev-down dev-logs migrate migrate-down jet-gen swagger sdk-types dev-api dev-auth dev-app dev smoke clean

BIN := ./bin
DATABASE_URL ?= postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable

# ==========================================
# Onboarding & First-Time Setup (DevEx)
# ==========================================

help:
	@echo "Zenv-sh Makefile Commands:"
	@echo ""
	@echo "--- Setup ---"
	@echo "  make boot           - First-time setup (envs, Go tools, build Docker images, start infra)"
	@echo "  make setup          - Install Go tools and copy .env files (no pnpm install)"
	@echo ""
	@echo "--- Local Development ---"
	@echo "  make dev            - Start all services (API, Auth, App) concurrently"
	@echo "  make dev-api        - Start the Go API (Portless)"
	@echo "  make dev-auth       - Start the Auth service (Portless)"
	@echo "  make dev-app        - Start the Web App (Portless)"
	@echo "  make dev-build      - Build Docker images (runs pnpm install inside containers)"
	@echo "  make dev-start      - Start Docker infra without rebuilding (fast)"
	@echo "  make dev-down       - Stop Docker infra"
	@echo "  make dev-logs       - Tail Docker logs"
	@echo ""
	@echo "--- Code Generation & Migrations ---"
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
	# @test -f apps/dashboard/.env || cp apps/dashboard/.env.example apps/dashboard/.env
	@test -f apps/identity/.env || cp apps/identity/.env.example apps/identity/.env
	@test -f cli/.env || cp cli/.env.example cli/.env 2>/dev/null || true

setup: init-env
	@echo "Installing Go tools (jet, swag)..."
	go install github.com/go-jet/jet/v2/cmd/jet@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ Go tools installed. (No host pnpm install – Docker handles Node dependencies.)"

boot: setup dev-build dev-start
	@echo "\n========================================"
	@echo "✅ Infrastructure and Database are ready!"
	@echo "========================================"
	@echo "All dependencies run inside Docker containers."
	@echo "To start developing, run: make dev"
	@echo "Or run these in separate terminals:"
	@echo "  make dev-api"
	@echo "  make dev-auth"
	@echo "  make dev-app"

# ==========================================
# Build (optional – for CI or production)
# ==========================================

all: build

build: build-api build-cli build-app build-auth build-docs build-packages

build-api:
	go build -o $(BIN)/zenv-api ./api/cmd/zenv-api

build-cli:
	go build -o $(BIN)/zenv ./cli/cmd/zenv

build-tf:
	go build -o $(BIN)/terraform-provider-zenv ./terraform-provider-zenv

build-auth:
	pnpm -C apps/identity run build

build-app:
	pnpm -C apps/dashboard run build

build-docs:
	pnpm -C apps/docs run build

build-packages:
	pnpm -r --filter './packages/*' run build

typecheck-packages:
	pnpm -r --filter './packages/*' run typecheck

lint-ts:
	pnpm lint

# --- Test ---
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
# Dev infrastructure (Docker)
# ==========================================

# Build images explicitly (use this after dependency changes)
dev-build:
	docker compose --profile local build

# Start containers – builds images if missing, otherwise uses cache
dev-up:
	docker compose --profile local up -d

# Alias for dev-up (fast start, builds only when necessary)
dev-start: dev-up

dev-down:
	docker compose --profile local down -v

dev-logs:
	docker compose --profile local logs -f --tail=200

# ==========================================
# Migrations – run on host (requires pnpm)
# ==========================================

migrate:
	pnpm -C apps/identity run db:migrate

migrate-down:
	pnpm -C apps/identity exec drizzle-kit drop

# ==========================================
# Code Generation
# ==========================================

jet-gen:
	~/go/bin/jet -dsn="$(DATABASE_URL)" -schema=public -path=./api/internal/store/gen

swagger:
	~/go/bin/swag init -g api/cmd/zenv-api/main.go -o api/docs --parseDependency --parseInternal

sdk-types: swagger
	pnpm exec swagger2openapi api/docs/swagger.json -o api/docs/openapi.json
	pnpm -C packages/sdk exec openapi-typescript ../../api/docs/openapi.json -o src/api.d.ts
	cp packages/sdk/src/api.d.ts apps/dashboard/src/lib/api.d.ts

# ==========================================
# Dev (Portless)
# ==========================================

dev-api:
	env $$(grep -v '^#' api/.env | xargs) portless api.zenv go run ./api/cmd/zenv-api

dev-auth:
	portless auth.zenv pnpm -C apps/identity run dev

dev-app:
	portless app.zenv pnpm -C apps/dashboard dev

dev:
	@make -j 3 dev-api dev-auth dev-app

# ==========================================
# Smoke tests & Clean
# ==========================================

smoke: build
	./tests/smoke.sh

clean:
	rm -rf $(BIN)

# --- Preview ---
preview: build
	$(BIN)/zenv-api &

# --- Prod infrastructure (local testing) ---
PROD_COMPOSE := -f docker-compose.yml
PROD_ENV_FILE ?= .env.prod

prod-up:
	@if [ -f "$(PROD_ENV_FILE)" ]; then \
		docker compose $(PROD_COMPOSE) --env-file $(PROD_ENV_FILE) up -d --remove-orphans; \
	else \
		echo "WARNING: $(PROD_ENV_FILE) not found, using current environment variables" >&2; \
		docker compose $(PROD_COMPOSE) up -d --remove-orphans; \
	fi

prod-down:
	docker compose $(PROD_COMPOSE) down -v

prod-recreate: prod-down prod-up

prod-logs:
	docker compose $(PROD_COMPOSE) --env-file $(PROD_ENV_FILE) logs -f --tail=200

prod-ps:
	docker compose $(PROD_COMPOSE) ps

prod-shell:
	docker compose $(PROD_COMPOSE) exec api sh