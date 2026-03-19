# Current Tasks

## Up Next

### API auth endpoints
- [ ] OAuth callback handler (GitHub, Google)
- [x] Session management (Redis-backed) — SessionManager with create/get/update/delete
- [x] Vault Key verification endpoint — POST /v1/auth/unlock (Auth Key hash comparison, returns wrapped DEK)
- [x] Account creation flow — POST /v1/auth/signup (stores salt, wrapped DEK, public key, auth_key_hash)
- [x] Dev login endpoint — POST /v1/auth/login (temporary, returns salt + vault_key_type)
- [x] Logout endpoint — POST /v1/auth/logout
- [x] RequireSession middleware (identity layer gate)
- [x] RequireVaultUnlocked middleware (both layers gate)
- [x] Smoke test passing: signup → login → unlock (wrong key correctly rejected)

### API secrets CRUD
- [x] POST /v1/secrets — store encrypted item (ciphertext + nonce + name_hash)
- [x] GET /v1/secrets/:nameHash — retrieve single secret with full ciphertext
- [x] POST /v1/secrets/bulk — bulk fetch by list of name_hashes (schema manifest)
- [x] GET /v1/secrets — list metadata only (name_hash, version, updated_at — never ciphertext)
- [x] PUT /v1/secrets/:nameHash — update with version bump, new nonce
- [x] DELETE /v1/secrets/:nameHash — hard delete
- [x] All endpoints behind RequireSession + RequireVaultUnlocked middleware
- [x] Smoke test passing: create → list → get → update (v2) → delete

### API service tokens
- [x] POST /v1/tokens — create scoped token (plaintext shown once, SHA-256 hashed before storage)
- [x] GET /v1/tokens — list tokens for a project (never exposes hash or plaintext)
- [x] DELETE /v1/tokens/:tokenID — revoke instantly (sets revoked_at)
- [x] Token auth middleware — Bearer token → SHA-256 hash lookup, revocation + expiry checks
- [x] RequireWrite middleware — rejects read-only tokens on write endpoints
- [x] SDK routes mounted at /v1/sdk/* — same secrets CRUD, token-authenticated
- [x] Smoke test: create token → SDK create secret → SDK list → revoke → rejected

### CLI implementation
- [ ] `zenv login` — browser OAuth flow + keyring storage
- [x] `zenv secrets set KEY VALUE` — encrypt via Amnesia, store on API (create or update)
- [x] `zenv secrets get KEY` — fetch ciphertext, decrypt locally, print value
- [x] `zenv secrets list` — show hashed names + version + updated_at (never values)
- [x] `zenv secrets delete KEY` — remove from server
- [x] `zenv run -- COMMAND` — bulk fetch, decrypt all, inject as env vars, exec child
- [x] `zenv check KEY [KEY...]` — bulk verify secrets exist, exit 1 if any missing (CI use case)
- [x] Project context resolution — .zenv file walk-up, ZENV_* env vars, --project/--env flags
- [x] HTTP client for SDK API endpoints
- [x] Crypto helpers wrapping Amnesia for secret payloads
- [x] End-to-end test: set → get → list → delete → run all passing

### Amnesia WASM
- [x] Create wasm/main.go bridge exporting all Amnesia functions
- [x] wasm/go.mod with amnesia dependency, added to go.work
- [x] Compiles with standard Go (verified)
- [x] TinyGo build target in Makefile (`make wasm`)
- [x] TinyGo 0.40.1 + Go 1.25 SDK — compiles to 339KB .wasm
- [ ] Test WASM in Node.js environment

## Done

### Monorepo scaffold
- [x] go.work with 3 modules (amnesia, api, cli)
- [x] Makefile, docker-compose.yml, .gitignore, pnpm-workspace.yaml
- [x] All modules compile from root

### Amnesia crypto engine
- [x] random.go — GenerateSalt, GenerateNonce, GenerateKey
- [x] derive.go — DeriveKeys (Argon2id, PIN/passphrase adaptive)
- [x] symmetric.go — Encrypt/Decrypt/WrapKey/UnwrapKey (AES-256-GCM)
- [x] hash.go — HashName (HMAC-SHA256), HashAuthKey (Argon2id)
- [x] asymmetric.go — GenerateKeypair/WrapWithPublicKey/UnwrapWithPrivateKey (X25519)
- [x] 31 tests passing

### API skeleton
- [x] Chi router with health check, graceful shutdown, request logging
- [x] Config loading from env vars
- [x] Stub route groups for v1 auth, secrets, tokens, projects

### CLI skeleton
- [x] Cobra root with global --project/--env flags
- [x] All subcommands stubbed: login, whoami, secrets, run, tokens, env, check

### Database + store layer
- [x] Initial SQL migration — users, linked_providers, organizations, projects, project_vault_keys, project_key_grants, vault_items, service_tokens, organization_members, audit_logs (monthly partitioned)
- [x] Docker Compose: Postgres 17 (port 5434) + Redis 7
- [x] Go-Jet codegen — type-safe models + SQL builders at api/internal/store/gen/
- [x] pgx connection pool wired into API startup
- [x] Redis client wired into API startup
- [x] Smoke test passing: Postgres connected, Redis connected, /health returns 200
- [x] Makefile targets: make migrate, make jet-gen (with default DATABASE_URL)
