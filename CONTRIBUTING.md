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
- `apps/identity/` — Auth server (BSL)
- `apps/dashboard/` — Dashboard (BSL)

## Development

- **Go**: Standard library conventions, `log/slog`, errors returned not panicked.
- **TypeScript**: Strict mode, ESM dual-build (via `tsup`), structured files.
- **Tests**: stdlib `testing` for Go, `vitest` for TS. Run `pnpm test` for TS, `make test` for Go.
- **Crypto**: Never add network/DB deps to `amnesia/`.

### TypeScript File Conventions
When contributing to TS packages, avoid monolithic files:
- `@zenv-sh/sdk`: Add validation/utils to `schema.ts` or `encoding.ts`. Define new error types in `errors.ts`.
- `@zenv-sh/vite-plugin`: Add secret fetching logic to `loader.ts`, polling logic to `watcher.ts`.

## Pull Requests

1. Fork and create a branch from `main`.
2. Write tests for new functionality.
3. Run `make test`, `make lint`, and `pnpm lint` before submitting.
4. **Changesets**: If modifying a public `@zenv-sh/*` package, run `pnpm changeset` and commit the generated markdown file. This tracks versions for our GitHub Packages release.
5. Keep PRs focused — one feature or fix per PR.

## Reporting Bugs

Open a GitHub issue with:
- Steps to reproduce
- Expected vs actual behavior
- zEnv version (`zenv --version`)

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## License

By contributing, you agree that your contributions will be licensed under the same license as the component you're contributing to (MIT for tools, BSL 1.1 for server components).
