# zEnv CLI

Go CLI for zEnv. Single binary, no runtime dependency, fast startup. Uses [Amnesia](../amnesia/) natively for client-side crypto — no WASM overhead.

## Commands

### Secrets

```bash
zenv secrets set KEY VALUE        # Encrypt and store a secret
zenv secrets get KEY              # Fetch, decrypt, print value
zenv secrets list                 # List metadata (never values)
zenv secrets delete KEY           # Remove from vault
```

### Run (secret injection)

```bash
zenv run -- node server.js        # Inject all secrets as env vars
zenv run -- python app.py         # Works with any command
```

### Check (CI validation)

```bash
zenv check DATABASE_URL STRIPE_KEY JWT_SECRET --env production
# Exits 1 if any secrets are missing — use in CI before deployment
```

### Environment

```bash
zenv env pull                     # Write secrets to .env.local
zenv env pull -o .env.production  # Custom output file
zenv env pull -o -                # Print to stdout
zenv env diff development production  # Compare environments
```

### Tokens

```bash
zenv tokens create --name ci-prod --permission read_write
zenv tokens list
zenv tokens revoke TOKEN_ID
```

### Auth

```bash
zenv login                        # Browser OAuth flow (not yet implemented)
zenv whoami                       # Show current auth context
```

## Configuration

The CLI resolves project context in this priority order:

| Priority | Source | Description |
| --- | --- | --- |
| 1 | `--project` / `--env` flags | Explicit on command — always wins |
| 2 | `.zenv` file in current or parent dir | Walk up tree like git |
| 3 | `ZENV_PROJECT` / `ZENV_ENV` env vars | Shell environment |
| 4 | Default | Falls back if only one project exists |

### .zenv file

```ini
# Committed to git — whole team inherits it
project=ecommerce-backend
env=development
```

### Environment variables

```bash
export ZENV_API_URL=http://localhost:8080   # API base URL
export ZENV_TOKEN=ze_development_...       # Service token
export ZENV_VAULT_KEY=...                   # Project Vault Key
export ZENV_PROJECT=<project-id>            # Project ID
export ZENV_ENV=development                 # Environment
```

## Install

```bash
# From source
make build-cli
./bin/zenv --help

# Or run directly
go run ./cli/cmd/zenv -- secrets list
```

Future distribution paths: `curl -fsSL https://zenv.sh/install | bash`, `npx zenv`, `brew install zenv`.

## How It Works

```
ZENV_VAULT_KEY
  → API: GET /sdk/projects/{id}/crypto → project_salt + wrapped_dek
  → Argon2id(ZENV_VAULT_KEY + project_salt) → Project KEK
  → AES-256-GCM unwrap(wrapped_dek, KEK) → Project DEK
  → All encrypt/decrypt uses Project DEK locally
  → Server never sees plaintext
```
