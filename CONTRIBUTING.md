# Contributing to zEnv

Thanks for your interest in contributing to zEnv!

## Getting Started

```bash
# Clone
git clone https://github.com/Judeadeniji/zenv-sh.git
cd zenv-sh

# Start Postgres + Redis
make dev-up

# Run migrations
make migrate

# Build
make build

# Run tests
make test
```

## Project Structure

- `amnesia/` — Go crypto engine (MIT)
- `api/` — Go API server (BSL)
- `cli/` — Go CLI (MIT)
- `packages/amnesia/` — TypeScript crypto engine (MIT)
- `packages/sdk/` — TypeScript SDK (MIT)
- `apps/auth/` — Auth server (BSL)
- `apps/web/` — Dashboard (BSL)

## Development

- Go: standard library conventions, `log/slog`, errors returned not panicked
- TypeScript: strict mode, ESM only
- Tests: stdlib `testing` for Go, vitest for TS
- Crypto: never add network/DB deps to `amnesia/`

## Pull Requests

1. Fork and create a branch from `main`
2. Write tests for new functionality
3. Run `make test` and `make lint` before submitting
4. Keep PRs focused — one feature or fix per PR

## Reporting Bugs

Open a GitHub issue with:
- Steps to reproduce
- Expected vs actual behavior
- zEnv version (`zenv --version`)

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## License

By contributing, you agree that your contributions will be licensed under the same license as the component you're contributing to (MIT for tools, BSL 1.1 for server components).
