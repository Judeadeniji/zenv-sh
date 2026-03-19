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
- [ ] POST /v1/secrets — store encrypted item (ciphertext + nonce + name_hash)
- [ ] GET /v1/secrets/:name_hash — retrieve single secret
- [ ] GET /v1/secrets — bulk fetch by list of name_hashes (schema manifest)
- [ ] PUT /v1/secrets/:name_hash — update (new version, new nonce)
- [ ] DELETE /v1/secrets/:name_hash — soft delete

### API service tokens
- [ ] POST /v1/tokens — create scoped token (project + env + permission)
- [ ] DELETE /v1/tokens/:id — revoke
- [ ] Token auth middleware — hash-based lookup, scope enforcement

### CLI implementation
- [ ] `zenv login` — browser OAuth flow + keyring storage
- [ ] `zenv secrets get/set/list` — call API, decrypt via Amnesia
- [ ] `zenv run` — fetch secrets, inject as env vars, exec child process
- [ ] `zenv check` — validate required secrets exist (CI use case)
- [ ] Project context resolution (.zenv file, --project/--env flags)

### Amnesia WASM
- [ ] Create wasm/main.go bridge exporting Amnesia functions
- [ ] TinyGo build target in Makefile
- [ ] Verify output <500KB
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
