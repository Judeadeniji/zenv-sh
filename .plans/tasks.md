# Current Tasks

## Up Next

### Better Auth + Developer Dashboard

- [ ] Migration 002: add better_auth_user_id to users, better_auth_org_id to organizations
- [ ] Scaffold apps/web/ — TanStack Start + Better Auth + Drizzle
- [ ] Better Auth config: GitHub + Google OAuth, org plugin, 2FA plugin
- [ ] Run Better Auth migrations (user, session, account, organization tables)
- [ ] Go API: better_auth.go middleware — read BA session from Postgres
- [ ] Go API: CORS middleware for app.zenv.sh
- [ ] Go API: POST /v1/auth/setup-vault — link BA user to zEnv crypto material
- [ ] Go API: GET /v1/auth/me — return user state (vault_setup_complete, salt, vault_key_type)
- [ ] Go API: modify /v1/auth/unlock to accept BA sessions
- [ ] Dashboard: vault-setup page (Amnesia TS in browser → crypto material → Go API)
- [ ] Dashboard: vault-unlock page (Vault Key → derive → unlock)
- [ ] Dashboard: _authed layout (redirect to login if no BA session)
- [ ] Dashboard: _unlocked layout (redirect to unlock if vault locked)
- [ ] Dashboard: secrets list/detail pages
- [ ] Dashboard: project switcher
- [ ] Dashboard: service token management
- [ ] Dashboard: organization + member management (BA org plugin)
- [ ] Org linkage: BA afterCreate hook syncs to zEnv organizations table
- [ ] Gate DevLogin behind ZENV_DEV_MODE env var
- [ ] Update smoke tests for BA auth flow

### @zenv/vite-plugin — build-time injection (Phase 2)

- [ ] Scaffold packages/vite-plugin
- [ ] Fetch + decrypt at build time via Amnesia TS
- [ ] Generate virtual `@zenv/secrets` module
- [ ] Secret leak prevention (server-only TS enforcement, runtime guard, post-build scan)

## Done

### Monorepo scaffold

- [x] go.work with 3 modules (amnesia, api, cli)
- [x] Makefile, docker-compose.yml, .gitignore, pnpm-workspace.yaml
- [x] All modules compile from root

### Amnesia crypto engine (Go)

- [x] random.go — GenerateSalt, GenerateNonce, GenerateKey
- [x] derive.go — DeriveKeys (Argon2id, PIN/passphrase adaptive)
- [x] symmetric.go — Encrypt/Decrypt/WrapKey/UnwrapKey (AES-256-GCM)
- [x] hash.go — HashName (HMAC-SHA256), HashAuthKey (Argon2id)
- [x] asymmetric.go — GenerateKeypair/WrapWithPublicKey/UnwrapWithPrivateKey (X25519 NaCl box)
- [x] EncryptWithNonce for deterministic test vector generation
- [x] 31 tests passing

### Amnesia TypeScript (pure TS reimplementation)

- [x] WASM approach dropped — TinyGo version conflicts, goroutine limitations
- [x] AES-256-GCM encrypt/decrypt/wrap/unwrap via Web Crypto API
- [x] Argon2id key derivation via hash-wasm
- [x] HMAC-SHA256 name hashing via Web Crypto API
- [x] X25519 asymmetric ops via tweetnacl (NaCl box — matches Go nacl/box)
- [x] GenerateSalt/GenerateNonce/GenerateKey via crypto.getRandomValues
- [x] Cross-language parity: shared test vectors (JSON), 30/30 passing

### Database + store layer

- [x] Initial SQL migration — users, linked_providers, organizations, projects, project_vault_keys, project_key_grants, vault_items, service_tokens, organization_members, audit_logs (monthly partitioned)
- [x] Docker Compose: Postgres 17 (port 5434) + Redis 7
- [x] Go-Jet codegen — type-safe models + SQL builders at api/internal/store/gen/
- [x] pgx connection pool + Redis client wired into API startup
- [x] Makefile targets: make migrate, make jet-gen

### API auth endpoints

- [x] Session management (Redis-backed) — SessionManager with create/get/update/delete
- [x] Vault Key verification — POST /v1/auth/unlock (Auth Key hash comparison, returns wrapped DEK)
- [x] Account creation — POST /v1/auth/signup (stores salt, wrapped DEK, public key, auth_key_hash)
- [x] Dev login — POST /v1/auth/login (temporary, returns salt + vault_key_type)
- [x] Logout — POST /v1/auth/logout
- [x] RequireSession + RequireVaultUnlocked middleware

### API secrets CRUD

- [x] POST /v1/secrets — store encrypted item
- [x] GET /v1/secrets/:nameHash — retrieve single secret
- [x] POST /v1/secrets/bulk — bulk fetch by name hashes (schema manifest)
- [x] GET /v1/secrets — list metadata only (never ciphertext)
- [x] PUT /v1/secrets/:nameHash — update with version bump
- [x] DELETE /v1/secrets/:nameHash — hard delete
- [x] SDK routes at /v1/sdk/* — token-authenticated mirror

### API service tokens

- [x] POST /v1/tokens — create scoped token (SHA-256 hashed before storage)
- [x] GET /v1/tokens — list tokens for a project
- [x] DELETE /v1/tokens/:tokenID — revoke instantly
- [x] Token auth middleware — Bearer token hash lookup + revocation/expiry checks
- [x] RequireWrite middleware — rejects read-only tokens on write endpoints

### API project CRUD + crypto

- [x] POST /v1/projects — create with client-generated crypto (transactional: project + vault key + key grant)
- [x] GET /v1/projects — list by organization
- [x] GET /v1/projects/{id} — get single project
- [x] GET /v1/sdk/projects/{id}/crypto — return salt + wrapped DEK for SDK key derivation

### OpenAPI + typed SDK client

- [x] Swag annotations on all API handlers
- [x] make swagger — generate Swagger 2.0 spec
- [x] make sdk-types — convert to OpenAPI 3.0 + generate TypeScript types
- [x] openapi-fetch client in SDK — fully typed from spec

### @zenv/sdk — TypeScript SDK

- [x] Scaffold packages/sdk with openapi-fetch + openapi-typescript
- [x] ZEnv class with load(), get(), set(), delete()
- [x] Browser ban — hard error if window is defined
- [x] initCrypto() wired to real GET /sdk/projects/{id}/crypto endpoint
- [x] Full zero-knowledge flow: ZENV_VAULT_KEY → Argon2id → Project KEK → unwrap DEK → encrypt/decrypt

### CLI implementation

- [x] `zenv secrets set/get/list/delete` — full encrypt/decrypt via Amnesia
- [x] `zenv run -- COMMAND` — inject secrets as env vars, exec child process
- [x] `zenv check KEY [KEY...]` — CI secret validation
- [x] Project context resolution — .zenv file walk-up, ZENV_* env vars, --project/--env flags
- [x] HTTP client for SDK API endpoints
- [x] Crypto wired to real project crypto endpoint (no more hardcoded salt)
- [x] `zenv tokens create/revoke/list` — wired to real API
- [x] `zenv env pull` — write secrets to .env file
- [x] `zenv env diff ENV1 ENV2` — compare environments

### Standard Schema support in SDK

- [x] Accept Zod, Valibot, ArkType schemas via Standard Schema v1 interface
- [x] Schema keys as fetch manifest
- [x] Validate + transform decrypted values
- [x] zenv() convenience factory function
- [x] strict mode + disableValidation option

### Publishing prep + CI

- [x] TS packages: exports, files, engines, build config for npm
- [x] tsconfig.build.json with rewriteRelativeImportExtensions
- [x] GitHub Actions CI: Go tests, TS tests, cross-language parity, cross-compile
- [x] .editorconfig, .nvmrc
- [x] READMEs for root + all 5 packages
