# zEnv

> ⚠️ **Pre-alpha. Not ready for use. Breaking changes happen without notice.**
> This is under active solo development. Do not use this to store real secrets yet.

**Zero-knowledge secret manager. Trust the math, not the server.**

zEnv is an encrypted vault for storing and sharing sensitive data API keys,
credentials, passwords where even we as the provider cannot read your data.
All encryption and decryption happens client-side via the Amnesia engine.
The server is a ciphertext warehouse.

## Status

| Component       | Status                                              |
| --------------- | --------------------------------------------------- |
| Amnesia (Go)    | ✅ Complete — cross-language parity tests passing   |
| Amnesia (TS)    | ✅ Complete                                         |
| API             | ✅ Complete                                         |
| Auth server     | ✅ Complete — dev auth only, OAuth not yet wired    |
| CLI             | ✅ Complete                                         |
| SDK             | ✅ Complete                                         |
| Dashboard       | 🚧 In progress                                      |
| Docs site       | 🚧 In progress                                      |
| Security audit  | ❌ Not done — do not use for real secrets           |
| OAuth           | ❌ Not done — dev login only                        |
| Production deploy | ❌ Not done                                       |

## What works right now

The core cryptographic layer and API are functional and tested. You can run
it locally, create projects, store secrets, and retrieve them via the CLI
and SDK. Cross-language parity between Go and TypeScript Amnesia is verified
by test vectors.

What does not exist yet: production deployment, OAuth, a finished dashboard,
a security audit, and anything resembling stability guarantees.

## What this is not yet

- Not audited by a third party
- Not production deployed
- Not stable — APIs will break
- Not accepting external contributions yet
- Not ready for storing secrets you care about

## Architecture

```
Go monorepo (go.work)
├── amnesia/              Pure cryptographic engine — Argon2id, AES-256-GCM, X25519 NaCl box
├── api/                  HTTP API server (Chi) — stores and retrieves ciphertext, never decrypts
└── cli/                  CLI tool (Cobra) — zenv command, uses Amnesia natively

TypeScript packages (pnpm workspaces)
├── packages/amnesia/     Pure TS reimplementation — byte-identical to Go, cross-language parity enforced
├── packages/sdk/         @zenv/sdk — typed API client + schema validation, zero crypto logic
└── apps/
    ├── auth/             Identity layer (TS) — handles authn for both the API and dashboard
    ├── web/              TanStack Start dashboard — manage projects, environments, and secrets
    └── docs/             Documentation site (Astro)
```

## Quick Start

```bash
# Start Postgres + Redis
make dev-up

# Run database migrations
make migrate

# Build binaries
make build

# Run the API server
DATABASE_URL="postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable" \
REDIS_URL="redis://localhost:6379" \
./bin/zenv-api

# Use the CLI
export ZENV_TOKEN=ze_...
export ZENV_PROJECT_KEY=...
export ZENV_PROJECT=<project-id>
export ZENV_ENV=development

./bin/zenv secrets set DATABASE_URL "postgres://prod:secret@db.internal/myapp"
./bin/zenv secrets get DATABASE_URL
./bin/zenv run -- node server.js
```

## SDK Usage

```typescript
import { zenv } from "@zenv/sdk";
import { z } from "zod";

const vault = zenv({
  token: process.env.ZENV_TOKEN!,
  projectKey: process.env.ZENV_PROJECT_KEY!,
  projectId: process.env.ZENV_PROJECT_ID!,
  environment: process.env.NODE_ENV,
  schema: z.object({
    STRIPE_API_KEY: z.string().min(1),
    DATABASE_URL: z.string().url(),
    PORT: z.string().transform(Number),
  }),
});

const secrets = await vault.load();
// secrets.STRIPE_API_KEY → string
// secrets.PORT → number (transformed)
```

## Project Structure

```
zEnv/
├── go.work                      Go workspace (3 modules)
├── Makefile                     Build, test, migrate, codegen
├── docker-compose.yml           Postgres 17 + Redis 7
├── amnesia/                     Pure Go crypto engine
│   ├── derive.go                Argon2id key derivation (KEK + Auth Key)
│   ├── symmetric.go             AES-256-GCM encrypt/decrypt/wrap/unwrap
│   ├── asymmetric.go            X25519 keypair + NaCl box
│   ├── hash.go                  HMAC-SHA256 name hashing, Auth Key hashing
│   └── random.go                Secure random generation
├── api/                         HTTP API server
│   ├── cmd/zenv-api/            Entrypoint
│   ├── internal/handler/        Auth, secrets, tokens, projects
│   ├── internal/middleware/      Session + token auth
│   ├── migrations/              SQL schema
│   └── docs/                    OpenAPI spec (auto-generated via swag)
├── cli/                         CLI tool
│   ├── cmd/zenv/                Entrypoint
│   └── internal/commands/       Cobra subcommands
├── packages/amnesia/            TypeScript crypto engine (Web Crypto API + hash-wasm)
├── packages/sdk/                @zenv/sdk (openapi-fetch + Standard Schema)
├── apps/
│   ├── auth/                        Identity server — separate authn layer for API + dashboard
│   ├── web/                         TanStack Start dashboard
│   └── docs/                        Astro documentation site
└── tests/                       Cross-language test vectors + smoke tests
```

## Key Hierarchy

```
Vault Key (user's memory — PIN or passphrase)
  │
  ▼ Argon2id — ONE run, 64-byte output
  ┌──────────────────────────────────┐
  │ bytes 0-31        bytes 32-63    │
  │ KEK               Auth Key       │
  │ (wraps DEK)       (server verify)│
  └──────────────────────────────────┘
        │
        ▼ AES-256-GCM unwrap
      DEK (Data Encryption Key)
        │
        ▼ AES-256-GCM per item
      Encrypted vault items (one per DB row)
```

The server stores only the wrapped DEK and ciphertext. It cannot derive the KEK (needs the Vault Key), cannot unwrap the DEK, and cannot decrypt any vault item.

## Make Targets

```bash
make build          # Build bin/zenv-api + bin/zenv
make test           # Run all Go tests
make test-amnesia   # Run Amnesia crypto tests
make dev-up         # Start Postgres + Redis
make dev-down       # Stop and remove containers
make migrate        # Run database migrations
make jet-gen        # Regenerate Go-Jet types from DB schema
make swagger        # Regenerate OpenAPI spec from swag annotations
make sdk-types      # Regenerate TypeScript types from OpenAPI spec
make lint           # Run golangci-lint
make smoke          # Run smoke tests against live API
```

## Packages

| Package                                | Description            | Docs                                   |
| -------------------------------------- | ---------------------- | -------------------------------------- |
| [amnesia/](amnesia/)                   | Go crypto engine       | [README](amnesia/README.md)            |
| [api/](api/)                           | Go HTTP API            | [README](api/README.md)                |
| [cli/](cli/)                           | Go CLI tool            | [README](cli/README.md)                |
| [packages/amnesia/](packages/amnesia/) | TypeScript crypto      | [README](packages/amnesia/README.md)   |
| [packages/sdk/](packages/sdk/)         | TypeScript SDK         | [README](packages/sdk/README.md)       |
| [apps/auth/](apps/auth/)               | Identity server (TS)   | —                                      |
| [apps/web/](apps/web/)                 | TanStack Start dashboard | —                                    |
| [apps/docs/](apps/docs/)               | Docs site (Astro)      | —                                      |

## License

Proprietary. See LICENSE for details.
