# zEnv API

Go HTTP API server. The ciphertext warehouse. Stores and retrieves encrypted blobs — participates in zero decryption.

Built with [Chi](https://github.com/go-chi/chi) router, [Go-Jet](https://github.com/go-jet/jet) for type-safe SQL, PostgreSQL, and Redis.

## Endpoints

### Auth

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/v1/auth/signup` | none | Create account with client-generated crypto material |
| POST | `/v1/auth/login` | none | Dev login by email (temporary — replaced by OAuth) |
| POST | `/v1/auth/unlock` | session | Verify Vault Key, return wrapped DEK |
| POST | `/v1/auth/logout` | session | Destroy session |

### Secrets (Dashboard)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/v1/secrets` | session+vault | Store encrypted secret |
| GET | `/v1/secrets` | session+vault | List metadata (never ciphertext) |
| GET | `/v1/secrets/:nameHash` | session+vault | Get single secret |
| POST | `/v1/secrets/bulk` | session+vault | Bulk fetch by name hashes |
| PUT | `/v1/secrets/:nameHash` | session+vault | Update (version auto-incremented) |
| DELETE | `/v1/secrets/:nameHash` | session+vault | Delete |

### Secrets (SDK/CLI)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/v1/sdk/secrets` | bearer token | Store encrypted secret |
| GET | `/v1/sdk/secrets` | bearer token | List metadata |
| GET | `/v1/sdk/secrets/:nameHash` | bearer token | Get single secret |
| POST | `/v1/sdk/secrets/bulk` | bearer token | Bulk fetch |
| PUT | `/v1/sdk/secrets/:nameHash` | bearer token (read_write) | Update |
| DELETE | `/v1/sdk/secrets/:nameHash` | bearer token (read_write) | Delete |

### Projects

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/v1/projects` | session+vault | Create project with crypto material |
| GET | `/v1/projects` | session+vault | List projects in org |
| GET | `/v1/projects/:id` | session+vault | Get single project |
| GET | `/v1/sdk/projects/:id/crypto` | bearer token | Get project salt + wrapped DEK |

### Service Tokens

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/v1/tokens` | session+vault | Create scoped token (shown once) |
| GET | `/v1/tokens` | session+vault | List tokens (never exposes hash) |
| DELETE | `/v1/tokens/:id` | session+vault | Revoke instantly |

## Dev Setup

```bash
# Start Postgres + Redis
make dev-up

# Run migrations
make migrate

# Run the server
DATABASE_URL="postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable" \
REDIS_URL="redis://localhost:6379" \
PORT=8080 \
./bin/zenv-api
```

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | required | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `PORT` | `8080` | HTTP listen port |
| `VERBOSE` | `false` | Enable debug logging (request logs) |

## Auth Model

Two independent auth mechanisms:

- **Session auth** (cookie `session_id`) — for dashboard (human access). Requires identity layer (OAuth) + vault unlock (Vault Key).
- **Bearer token auth** (`Authorization: Bearer ze_...`) — for SDK/CLI (machine access). Token is SHA-256 hashed before storage. Scoped to project + environment + permission.

## OpenAPI Spec

Auto-generated from [swag](https://github.com/swaggo/swag) annotations in handler files.

```bash
make swagger        # Regenerate spec
make sdk-types      # Regenerate TypeScript types from spec
```

Spec files: `api/docs/swagger.json` (Swagger 2.0), `api/docs/openapi.json` (OpenAPI 3.0).
