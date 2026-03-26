# Current Tasks

## Up Next

### Fixes + Polish

- [ ] Audit log query — Go-Jet JOIN generates `uuid = text` operator mismatch (needs raw SQL cast)
- [ ] Argon2id in Web Worker — prevent UI freeze during unlock on slow pins
- [ ] Auth pages protected from authenticated users (redirect /login → /unlock if already authed)
- [ ] Blob encryption — encrypt arbitrary payloads (files, notes, cards) per the plan doc

### Deploy SaaS

- [ ] Neon Postgres + Upstash Redis accounts
- [ ] Fly.io apps (api + auth)
- [ ] Cloudflare DNS for zenv.sh
- [ ] Vercel project for dashboard
- [ ] GitHub Actions deploy pipeline

### Encryption API

- [ ] New API routes: /v1/encrypt, /v1/decrypt, /v1/vaults
- [ ] Usage metering middleware
- [ ] SDK methods: zenv.encrypt(data), zenv.decrypt(ciphertext)
- [ ] Encryption API docs

### Docs site

- [ ] Deploy to Cloudflare Pages
- [ ] Encryption API docs (after API is built)

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
- [x] Migration 002: identity_id on users, identity_org_id on organizations
- [x] Migration 004: renamed better_auth columns to generic identity names
- [x] Migration 005: dek_version on vault_items + project_vault_keys, project_rotations, vault_item_rotations tables
- [x] Migration 006: users.preferences jsonb column
- [x] Docker Compose: Postgres 17 (port 5434) + Redis 7
- [x] Go-Jet codegen — type-safe models + SQL builders at api/internal/store/gen/
- [x] pgx connection pool + Redis client wired into API startup
- [x] Makefile targets: make migrate, make jet-gen

### Auth server (standalone)

- [x] Scaffold apps/auth/ — Hono + Drizzle on Postgres
- [x] Email/password auth enabled
- [x] Admin plugin (user roles, admin dashboard)
- [x] Organization plugin with afterCreate hooks syncing to zEnv tables
- [x] OpenAPI plugin (Scalar docs at /api/auth/reference)
- [x] @hono/node-server runtime, env validated via @t3-oss/env-core + Zod
- [x] Drizzle migrations for identity tables (user, session, account, verification, organization, member, invitation)
- [x] Cross-subdomain cookie support for production (.zenv.sh)
- [x] GitHub + Google OAuth (conditional on env vars)
- [x] 2FA plugin (twoFactor) with migration
- [x] Branded token prefix: ze_{env}_{random}
- [x] Removed all "Better Auth" / "BA" naming — generic identity layer

### Go API — vault + identity integration

- [x] IdentitySession middleware — reads session cookie from Postgres, resolves zEnv user
- [x] Bearer token fallback for cross-origin API calls
- [x] POST /v1/auth/setup-vault — link identity to zEnv crypto material
- [x] GET /v1/auth/me — return vault setup/unlock state
- [x] POST /v1/auth/unlock — verify Auth Key, return wrapped DEK
- [x] CORS middleware via go-chi/cors (CORS_ORIGINS env var)
- [x] Removed legacy auth (signup, DevLogin, logout, SessionManager)
- [x] API is vault-only — identity handled by standalone auth server
- [x] Regenerated OpenAPI spec
- [x] Smoke tests rewritten for auth server + vault-only API architecture
- [x] GET /v1/preferences + PUT /v1/preferences — server-synced user preferences (jsonb)
- [x] Two-phase DEK rotation: start/stage/commit/cancel endpoints
- [x] All handlers documented with Swagger/OpenAPI annotations
- [x] Pretty slog TUI handler for readable terminal output

### API secrets CRUD

- [x] POST /v1/secrets — store encrypted item
- [x] GET /v1/secrets/:nameHash — retrieve single secret
- [x] POST /v1/secrets/bulk — bulk fetch by name hashes (schema manifest)
- [x] GET /v1/secrets — list metadata only (never ciphertext)
- [x] PUT /v1/secrets/:nameHash — update with version bump
- [x] DELETE /v1/secrets/:nameHash — hard delete
- [x] SDK routes at /v1/sdk/* — token-authenticated mirror
- [x] Server-side filtering, sorting, and pagination across all list endpoints

### API service tokens

- [x] POST /v1/tokens — create scoped token (SHA-256 hashed before storage)
- [x] GET /v1/tokens — list tokens for a project
- [x] DELETE /v1/tokens/:tokenID — revoke instantly
- [x] Token auth middleware — Bearer token hash lookup + revocation/expiry checks
- [x] RequireWrite middleware — rejects read-only tokens on write endpoints
- [x] Branded token prefix: ze_{env}_{random}

### API project CRUD + crypto

- [x] POST /v1/projects — create with client-generated crypto (transactional: project + vault key + key grant)
- [x] GET /v1/projects — list by organization
- [x] GET /v1/projects/{id} — get single project
- [x] GET /v1/projects/{id}/stats — secret/token/audit counts by environment
- [x] DELETE /v1/projects/{id} — delete project + all secrets/tokens/grants
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

### CLI

- [x] `zenv secrets set/get/list/delete` — full encrypt/decrypt via Amnesia
- [x] `zenv run -- COMMAND` — inject secrets as env vars, exec child process
- [x] `zenv check KEY [KEY...]` — CI secret validation
- [x] `zenv tokens create/revoke/list` — wired to real API
- [x] `zenv env pull` — write secrets to .env file
- [x] `zenv env diff ENV1 ENV2` — compare environments
- [x] `zenv projects init/list/get/create` — full client-side crypto
- [x] `zenv orgs create/list/get/members/add-member/remove-member`
- [x] `zenv login` — prompt for service token, store to credentials
- [x] `zenv whoami` — show auth context, project, environment, validate token
- [x] `zenv config set/get/list/unset/path` — git-style config (local default, --global flag)
- [x] File-based config: ~/.config/zenv/config + credentials + local .zenv
- [x] Config resolution: flags → local .zenv → global config → env vars → defaults

### Standard Schema support in SDK

- [x] Accept Zod, Valibot, ArkType schemas via Standard Schema v1 interface
- [x] Schema keys as fetch manifest
- [x] Validate + transform decrypted values
- [x] zenv() convenience factory function
- [x] strict mode + disableValidation option

### Test suites

- [x] testutil package: Postgres + Redis testcontainers, migration runner, identity table DDL
- [x] Test fixtures: CreateIdentityUser, CreateZenvUser, CreateProject, CreateServiceToken (real Amnesia crypto)
- [x] httptest server helper with production router
- [x] Middleware tests (13): identity session (cookie, Bearer, expired, missing, invalid), vault lock/unlock, token auth (valid, invalid, prefix, revoked), read/write permissions
- [x] E2E integration tests (3): full secret lifecycle, read-only token rejection, invalid token rejection
- [x] Handler tests (37): auth (10), secrets (12), tokens (3), projects (5), orgs (7)
- [x] CLI config tests (9): defaults, env overrides, flags, global/local files, credentials perms, set/unset
- [x] All 93 tests passing via `make test`

### Packaging + OSS

- [x] Dockerfile for API (multi-stage Go build)
- [x] Dockerfile for auth server (Node.js)
- [x] docker-compose.prod.yml for self-hosting (API + auth + Postgres + Redis)
- [x] .env.example for production config
- [x] BSL 1.1 license for server components (API, auth, dashboard)
- [x] MIT license for tools (CLI, SDK, Amnesia)
- [x] CONTRIBUTING.md + SECURITY.md
- [x] Updated roadmap with encryption API and business model

### Docs site (Starlight)

- [x] Scaffold apps/docs/ with Astro Starlight
- [x] Introduction, quickstart, how-it-works (architecture explainer)
- [x] CLI reference: installation, configuration, all commands
- [x] SDK reference: installation, usage with schema validation
- [x] Self-hosting guide with docker-compose
- [x] 11 pages, Pagefind search, all content from actual codebase
- [x] Amnesia crypto engine docs: overview, key derivation, symmetric, hashing, asymmetric, random (6 pages, Go + TS examples)

### Publishing prep + CI

- [x] TS packages: exports, files, engines, build config for npm
- [x] tsconfig.build.json with rewriteRelativeImportExtensions
- [x] GitHub Actions CI: Go tests, TS tests, cross-language parity, cross-compile
- [x] .editorconfig, .nvmrc
- [x] READMEs for root + all 5 packages

### Developer Dashboard (TanStack Start)

- [x] Scaffold apps/web/ — TanStack Start + Tailwind + shadcn/ui
- [x] Auth flow — login, signup, vault setup, unlock (with PIN + passphrase)
- [x] Protected layouts — `_authed` (session) + `_unlocked` (crypto keys in memory)
- [x] Vault unlock gating — beforeLoad throws redirect; queries disabled until crypto present
- [x] Org dashboard — quick actions, stat cards (projects, members), project list, team preview
- [x] Project dashboard — environment breakdown, stats grid, project key reveal, quick start, recent activity, token overview
- [x] Project settings — general info, project key reveal, DEK rotation trigger, danger zone (type-to-confirm delete)
- [x] Secrets page — list with client-side decrypt, search/filter/sort, create, edit, version history, rollback, delete
- [x] Service tokens page — list, create (scoped to env + permission), revoke
- [x] Audit log page — server-side paginated DataTable with filters
- [x] Members page — list, invite, remove, role management
- [x] Org settings page — general info, danger zone
- [x] Account settings — profile, security, linked accounts
- [x] App sidebar — org switcher, project list, pinned projects (hover to pin/unpin), project + org nav
- [x] App header — breadcrumbs, environment switcher (Dev/Stg/Prod)
- [x] Server-synced preferences — active environment + pinned projects persisted to users.preferences
- [x] DEK rotation dialog — multi-step UI (confirm → progress → complete/error) using two-phase rotation API
- [x] Route files reorganized to `*/route.tsx` convention
- [x] Portless local dev — .localhost domains, cross-subdomain cookies
