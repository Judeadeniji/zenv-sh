# zEnv Dashboard

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](../../LICENSE-AGPL)

Web dashboard for **zEnv** — the zero-knowledge secret manager. Built with [TanStack Start](https://tanstack.com/start), React 19, and Tailwind CSS v4.

## Stack

| Layer | Technology |
|-------|-----------|
| Framework | [TanStack Start](https://tanstack.com/start) (SSR + SPA) |
| UI | React 19, [shadcn/ui](https://ui.shadcn.com), [Base UI](https://base-ui.com) |
| Routing | [TanStack Router](https://tanstack.com/router) (file-based) |
| Data | [TanStack Query](https://tanstack.com/query), [openapi-fetch](https://github.com/openapi-ts/openapi-fetch) |
| Styling | [Tailwind CSS v4](https://tailwindcss.com) |
| Forms | [React Hook Form](https://react-hook-form.com) + [Zod](https://zod.dev) |
| Auth | [Better Auth](https://better-auth.com) (client) |
| Crypto | [@zenv-sh/amnesia](../../packages/amnesia/) (client-side encryption) |
| Server | [Nitro](https://nitro.build) |
| Linting | [Biome](https://biomejs.dev) |
| Testing | [Vitest](https://vitest.dev) + [Testing Library](https://testing-library.com) |

## Development

```bash
# From the repo root
pnpm install

# Start the dashboard dev server
cd apps/dashboard
pnpm dev
```

The dev server starts at `http://localhost:3000` by default.

### Environment Variables

Create a `.env.dev` file (see `.env.dev` for reference):

| Variable | Description |
|----------|-------------|
| `VITE_API_URL` | zEnv API server URL |
| `VITE_AUTH_URL` | Identity/auth server URL |

## Building

```bash
pnpm build
```

Produces a Nitro server bundle in `.output/`.

## Testing

```bash
pnpm test
```

## Project Structure

```
apps/dashboard/
├── src/
│   ├── routes/          # File-based routes (TanStack Router)
│   ├── components/      # Shared UI components
│   └── lib/             # Utilities, API client, auth
├── public/              # Static assets
├── vite.config.ts       # Vite + TanStack Start config
└── components.json      # shadcn/ui configuration
```

## Security

All cryptographic operations (vault unlock, secret encrypt/decrypt, key derivation) happen **in the browser** using `@zenv-sh/amnesia`. The dashboard never sends plaintext secrets to the server.

## License

AGPL-3.0 — see [LICENSE](../../LICENSE-AGPL).
