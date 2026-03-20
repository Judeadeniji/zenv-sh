.PHONY: all build test lint clean dev-up dev-down dev-api dev-auth migrate jet-gen swagger sdk-types auth-build migrate-auth

BIN := ./bin
DATABASE_URL ?= postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable

all: build

# --- Build ---
build: build-api build-cli

build-api:
	go build -o $(BIN)/zenv-api ./api/cmd/zenv-api

build-cli:
	go build -o $(BIN)/zenv ./cli/cmd/zenv

# --- Test ---
test:
	go test ./amnesia/... ./api/... ./cli/...

test-amnesia:
	go test -v -count=1 ./amnesia/...

test-api:
	go test -v -count=1 ./api/...

test-cli:
	go test -v -count=1 ./cli/...

# --- Lint ---
lint:
	golangci-lint run ./amnesia/... ./api/... ./cli/...

# --- Dev infrastructure ---
dev-up:
	docker compose up -d

dev-down:
	docker compose down -v

# --- Migrations ---
migrate:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path ./api/migrations \
		-database "$(DATABASE_URL)" up

migrate-down:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path ./api/migrations \
		-database "$(DATABASE_URL)" down 1

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

# --- Dev (Portless) ---
dev-api:
	env $$(grep -v '^#' api/.env | xargs) portless zenv go run ./api/cmd/zenv-api

dev-auth:
	portless zenv-auth pnpm -C apps/auth run dev

# --- Auth server (Better Auth) ---
auth-build:
	pnpm -C apps/auth run build

migrate-auth:
	pnpm -C apps/auth run db:migrate

# --- Smoke tests (requires API running + Postgres + Redis) ---
smoke: build
	./tests/smoke.sh

# --- Clean ---
clean:
	rm -rf $(BIN)
