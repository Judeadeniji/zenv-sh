# CLAUDE.md

## Project

zEnv — zero-knowledge secret manager. All encryption/decryption happens client-side via the Amnesia engine. The server is a ciphertext warehouse.

## Repo Layout

- `amnesia/` — Pure Go crypto engine. **ZERO** network, DB, or config deps. Only `golang.org/x/crypto`.
- `api/` — Go HTTP API server (Chi). Stores and retrieves ciphertext. Never decrypts.
- `cli/` — Go CLI (Cobra). Uses Amnesia natively for client-side crypto.
- `packages/amnesia/` — Pure TypeScript reimplementation of Amnesia (Web Crypto API + hash-wasm + @noble/curves). Must produce byte-identical output to Go Amnesia — parity enforced via shared test vectors in CI.
- `packages/sdk/` — @zenv/sdk (future). Thin wrapper: API calls + schema validation + packages/amnesia for crypto.
- `packages/vite-plugin/` — @zenv/vite-plugin (future).
- `apps/` — TanStack Start dashboards (future).

Three separate Go modules wired via `go.work`. Build all with:
```
go build ./amnesia/... ./api/... ./cli/...
```

## Build Commands

```bash
make build          # Build bin/zenv-api + bin/zenv
make test-amnesia   # Run Amnesia tests
make test           # Run all tests
make dev-up         # Start Postgres + Redis (Docker Compose)
make dev-down       # Stop and remove containers
make migrate        # Run database migrations (requires DATABASE_URL)
make lint           # Run golangci-lint
```

## Critical Rules

1. **Amnesia purity** — Never add network, database, filesystem, or config imports to `amnesia/`. It takes bytes in and gives bytes out. If you need to import `net/http` or `database/sql` into amnesia, something is wrong.
2. **Zero-knowledge invariant** — The DEK, KEK, and Vault Key never leave the client. The API never decrypts user data. Never write server-side code that accesses plaintext secrets.
3. **Browser ban** — ZENV_TOKEN and ZENV_VAULT_KEY are server credentials. The SDK must throw a hard error if `window` is defined.
4. **Argon2id runs once** — One run per unlock, 64-byte output split: bytes 0-31 → KEK, bytes 32-63 → Auth Key. Never run it twice.
5. **Item-per-row storage** — Each secret is its own encrypted DB row. Never use a single-blob vault model.
6. **Cross-language parity** — Go Amnesia and TypeScript Amnesia must produce byte-identical outputs. Enforced by shared JSON test vectors in CI.

## Amnesia API

```go
DeriveKeys(vaultKey string, salt []byte, keyType KeyType) (kek, authKey []byte)
Encrypt(plaintext, key []byte) (ciphertext, nonce []byte, err error)
Decrypt(ciphertext, nonce, key []byte) (plaintext []byte, err error)
WrapKey(dek, kek []byte) (ciphertext, nonce []byte, err error)
UnwrapKey(ciphertext, nonce, kek []byte) (dek []byte, err error)
HashName(name string, hmacKey []byte) []byte
HashAuthKey(authKey []byte) []byte
GenerateKeypair() (publicKey, privateKey []byte, err error)
WrapWithPublicKey(payload, publicKey []byte) ([]byte, error)
UnwrapWithPrivateKey(ciphertext, privateKey []byte) ([]byte, error)
GenerateSalt() []byte
GenerateNonce() []byte
GenerateKey() []byte
```

## Licensing

Dual-licensed. See root `LICENSE` for the full mapping.

- **AGPL-3.0** — `api/`, `apps/auth/`, `apps/web/` (core API, auth server, dashboard)
- **MIT** — everything else (`amnesia/`, `cli/`, `packages/`, `apps/docs/`, `apps/consumer/`)

Each directory has its own LICENSE file. New directories default to MIT unless they are part of the core hosted platform.

## Style

- Go: standard library conventions, `log/slog` for logging, errors returned not panicked (except crypto/rand failure).
- SQL: migrations written by hand, Go-Jet for type-safe query building (codegen from DB schema).
- Config: environment variables only (12-factor). No YAML/TOML config files.
- Tests: stdlib `testing` package. Known-answer tests for crypto operations.
