.PHONY: all build test lint clean dev-up dev-down migrate jet-gen wasm

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

# --- WASM (requires TinyGo + Go 1.25 SDK at ~/sdk/go1.25.0) ---
wasm:
	cd wasm && GOWORK=off GOROOT=$(HOME)/sdk/go1.25.0 GOTOOLCHAIN=local PATH=$(HOME)/sdk/go1.25.0/bin:$(PATH) \
		tinygo build -o amnesia.wasm -target wasm -no-debug .

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

# --- Smoke tests (requires API running + Postgres + Redis) ---
smoke: build
	./tests/smoke.sh

# --- Clean ---
clean:
	rm -rf $(BIN) wasm/amnesia.wasm
