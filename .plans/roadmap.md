# zEnv Roadmap

## Phase 1 — Something That Works

Goal: A developer can use zEnv for a real project end-to-end.

- [x] Amnesia crypto engine (Go) — fully implemented and tested
- [x] Monorepo scaffold — go.work, Makefile, docker-compose
- [x] API server skeleton (Go, Chi) — health check, graceful shutdown
- [x] CLI skeleton (Go, Cobra) — all subcommands stubbed
- [ ] Amnesia TinyGo WASM build — compile to <500KB for TS SDK
- [ ] Database migrations — users, vault_items, projects, service_tokens, audit_logs
- [ ] API auth endpoints — OAuth callbacks, session management, Vault Key verification
- [ ] API secrets CRUD — encrypted item storage with HMAC name lookup
- [ ] API service tokens — create, scope, revoke (hashed before storage)
- [ ] API audit log pipeline — Redis buffer → Postgres bulk insert
- [ ] CLI implementation — login, secrets get/set/list, run, tokens
- [ ] @zenv/sdk — TypeScript WASM wrapper, schema-driven load(), typed returns
- [ ] Developer dashboard (TanStack Start) — basic, functional
- [ ] Documentation — quickstart + API reference
- [ ] Docker Compose local dev — Postgres + Redis running

## Phase 2 — Something Developers Will Adopt

Goal: Developers choose zEnv over alternatives and integrate it into their pipeline.

- [ ] GitHub Actions integration — published Action on Marketplace
- [ ] Audit logs — Redis buffer + partitioned Postgres, dashboard viewer
- [ ] Webhooks — register endpoints, HMAC-signed delivery, retry on failure
- [ ] Secret versioning and rollback
- [ ] @zenv/vite-plugin — build-time injection for edge runtimes
- [ ] Python SDK
- [ ] Full documentation site with framework guides
- [ ] Status page (status.zenv.sh)
- [ ] Discord community

## Phase 3 — Something Teams Will Pay For

Goal: Close first paying team customers. Public launch.

- [ ] RBAC — Admin / Senior Dev / Dev / Contractor / CI Bot roles
- [ ] Go SDK (shares Amnesia natively)
- [ ] More CI/CD integrations — GitLab, Docker, Kubernetes
- [ ] Secret rotation with expiry alerts
- [ ] SOC 2 audit process started
- [ ] Security whitepaper published
- [ ] Stripe billing integration
- [ ] Public launch

## Phase 4 — Something Enterprises Will Approve

Goal: Land first enterprise accounts.

- [ ] SSO / SAML integration
- [ ] SOC 2 Type II report completed
- [ ] Third-party security audit of Amnesia published
- [ ] Bug bounty program
- [ ] Uptime SLA (99.9% Team, 99.99% Enterprise)
- [ ] Advanced auto-rotation (Stripe, AWS, Twilio)
- [ ] On-premise deployment option

## Phase 5 — Consumer Product

Goal: Launch consumer product after developer product is stable.

- [ ] Browser extension (Chrome, Firefox, Safari, Edge)
- [ ] iOS and Android mobile apps
- [ ] Consumer dashboard — passwords, cards, notes, TOTP
- [ ] Breach monitor (HaveIBeenPwned)
- [ ] Secure sharing (one-time view links)
- [ ] Family plan

## Phase 6 — zEnv Uses zEnv

Goal: Dogfood — migrate all internal secrets to zEnv.

- [ ] Bootstrap: one root secret in AWS KMS authenticates with zEnv
- [ ] All infrastructure secrets managed by zEnv
- [ ] Publish "How zEnv Secures zEnv" blog post
