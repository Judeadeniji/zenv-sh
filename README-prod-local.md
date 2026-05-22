# Local Production-like Infra for zEnv

This guide helps you run a local stack that closely matches production, using Docker Compose and Caddy for HTTPS and domain routing.

## 1. Prerequisites
- Docker & Docker Compose
- Caddy (for local CA trust, run `caddy trust` if needed)
- pnpm (for JS/TS builds)

## 2. Setup

### 2.1. Environment Variables
Copy the example file and fill in secrets:

```sh
cp .env.prod.example .env.prod
# Edit .env.prod and set strong secrets for *_SECRET, *_PASSWORD, etc.
```

### 2.2. Build Artifacts
Build all containers and JS/TS apps:

```sh
make build
```

### 2.3. Start the Stack

```sh
make prod-up
```

- This uses `docker-compose.yml` + `docker-compose.prod.yml` for a full stack: Postgres, Redis, API, Auth, Web, and Caddy reverse proxy.
- Caddy will serve HTTPS for the local domains you set in `.env.prod` (e.g. `app.zenv.localhost`).

### 2.4. Check Status

```sh
make prod-ps
make prod-logs
```

### 2.5. Run Smoke Tests

```sh
make smoke
```

## 3. Accessing Services
- Web: https://app.zenv.localhost
- Auth: https://auth.zenv.localhost
- API: https://api.zenv.localhost/v1

## 4. Stopping

```sh
make prod-down
```

## 5. Notes
- For local HTTPS, trust the Caddy local CA: `caddy trust`
- You can override any variable in `.env.prod` as needed.
- For real prod, use real domains and secure secrets.

---

For troubleshooting, see logs with `make prod-logs` or open a shell in a service with `make prod-shell`.
