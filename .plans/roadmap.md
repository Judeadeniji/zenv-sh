# zEnv Roadmap

## Phase 1 — Something That Works

Goal: A developer can use zEnv for a real project end-to-end.

- [x] Amnesia crypto engine (Go + TypeScript, cross-language parity)
- [x] Monorepo scaffold — go.work, pnpm workspaces, Makefile, docker-compose
- [x] API server — secrets CRUD, service tokens, projects, orgs, audit logs
- [x] Auth server — standalone identity server (email/password, OAuth, 2FA, orgs)
- [x] CLI — secrets, tokens, projects, orgs, login, whoami, config
- [x] @zenv/sdk — TypeScript SDK with Standard Schema support
- [x] 93 tests (handler, middleware, E2E integration, CLI config)
- [x] Developer dashboard (TanStack Start)
- [x] DEK rotation — two-phase re-encryption (API + web UI)
- [x] Server-synced user preferences (active environment, pinned projects)
- [x] Dockerfiles + self-hosting guide
- [x] Documentation site (Starlight)
- [ ] Encryption API — encrypt/decrypt any data, not just secrets
- [ ] Deploy SaaS (Fly.io + Neon + Upstash + Vercel)

## Phase 2 — Something Developers Will Adopt

Goal: Developers choose zEnv over alternatives and integrate it into their pipeline.

- [ ] @zenv/vite-plugin — build-time secret injection
- [ ] GitHub Actions integration — published Action on Marketplace
- [ ] Webhooks — HMAC-signed delivery, retry on failure
- [ ] Python SDK
- [ ] Go SDK (shares Amnesia natively)
- [ ] Full documentation site with framework guides
- [ ] Status page (status.zenv.sh)
- [ ] Discord community

## Phase 3 — Something Teams Will Pay For

Goal: Close first paying customers. Public launch.

- [ ] Billing integration (LemonSqueezy or Stripe)
- [ ] Plan limits enforcement (free/pro/enterprise)
- [ ] Usage metering for encryption API
- [ ] Secret rotation with expiry alerts
- [ ] SOC 2 audit process started
- [ ] Security whitepaper published
- [ ] Public launch

## Phase 4 — Something Enterprises Will Approve

Goal: Land first enterprise accounts.

- [ ] SSO / SAML integration
- [ ] SOC 2 Type II report completed
- [ ] Third-party security audit of Amnesia
- [ ] Bug bounty program
- [ ] Uptime SLA (99.9% Team, 99.99% Enterprise)
- [ ] Advanced auto-rotation (Stripe, AWS, Twilio)
- [ ] On-premise deployment support

## Phase 5 — Consumer Product (zEnv Vault, closed-source)

Goal: Launch consumer product on the same Amnesia engine.

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
