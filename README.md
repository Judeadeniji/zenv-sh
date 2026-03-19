# zEnv

Zero-knowledge secret manager. Even we can't read your data.

## Architecture

```
amnesia/   Pure cryptographic engine (Go) — Argon2id, AES-256-GCM, X25519
api/       Core API server (Go, Chi) — the ciphertext warehouse
cli/       CLI tool (Go, Cobra) — zenv command
```

All encryption and decryption happens client-side via Amnesia. The server only stores ciphertext.

## Quick Start

```bash
# Start Postgres + Redis
make dev-up

# Build binaries
make build

# Run tests
make test-amnesia

# Run the API (requires DATABASE_URL)
DATABASE_URL="postgres://zenv:zenv_dev@localhost:5432/zenv?sslmode=disable" ./bin/zenv-api

# Use the CLI
./bin/zenv --help
```

## Project Structure

```
zEnv/
├── go.work                  Go workspace (3 modules)
├── amnesia/                 Pure crypto — zero network, zero DB deps
│   ├── derive.go            Argon2id key derivation (KEK + Auth Key)
│   ├── symmetric.go         AES-256-GCM encrypt/decrypt/wrap/unwrap
│   ├── asymmetric.go        X25519 keypair + NaCl box
│   ├── hash.go              HMAC-SHA256 name hashing, Auth Key hashing
│   └── random.go            Secure random generation
├── api/                     HTTP API server
│   ├── cmd/zenv-api/        Entrypoint
│   ├── internal/            Config, handlers, middleware, store
│   └── migrations/          SQL migrations
├── cli/                     CLI tool
│   ├── cmd/zenv/            Entrypoint
│   └── internal/commands/   Cobra subcommands
├── wasm/                    TinyGo WASM build output (future)
├── packages/                TypeScript packages (future)
└── apps/                    TanStack Start dashboards (future)
```

## Key Hierarchy

```
Vault Key (user's memory)
  → Argon2id (one run, 64 bytes)
    → bytes 0-31:  KEK (wraps the DEK)
    → bytes 32-63: Auth Key (server verification)
      → KEK unwraps → DEK (encrypts every vault item)
```

## License

Proprietary. See LICENSE for details.
