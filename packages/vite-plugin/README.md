# @zenv/vite-plugin

Vite plugin for [zEnv](https://zenv.sh). Fetches encrypted secrets at build time, decrypts them in Node, and injects the values into `import.meta.env` — without ever exposing your credentials to the browser.

## How it works

```
vite build / vite dev
  │
  └─ @zenv/vite-plugin (Node, build host)
       │
       ├─ ZENV_TOKEN + ZENV_PROJECT_KEY → @zenv/sdk → decrypt secrets
       │                                                   (stays in Node)
       └─ plaintext values → Vite define API → import.meta.env.ZENV_*
                                                   (lands in bundle)
```

`ZENV_TOKEN` and `ZENV_PROJECT_KEY` are never in the bundle. Only the decrypted values you declare in `schema` are injected.

## Install

```bash
pnpm add -D @zenv/vite-plugin
```

## Quick start

```ts
// vite.config.ts
import { defineConfig } from "vite"
import { z } from "zod"
import { zenvPlugin } from "@zenv/vite-plugin"

export default defineConfig({
  plugins: [
    zenvPlugin({
      token:      process.env.ZENV_TOKEN!,
      projectKey: process.env.ZENV_PROJECT_KEY!,
      projectId:  process.env.ZENV_PROJECT_ID!,
      environment: "production",   // or "development" | "staging"
      schema: z.object({
        STRIPE_PUBLISHABLE_KEY: z.string().startsWith("pk_"),
        API_BASE_URL:           z.string().url(),
      }),
    }),
  ],
})
```

In your app:

```ts
// Fully typed. Injected at build time. No network call from the browser.
const stripe = import.meta.env.ZENV_STRIPE_PUBLISHABLE_KEY
const api    = import.meta.env.ZENV_API_BASE_URL
```

## Options

| Option        | Type                            | Default                  | Description |
|---------------|---------------------------------|--------------------------|-------------|
| `token`       | `string`                        | **required**             | Service token (`ZENV_TOKEN`) |
| `projectKey`  | `string`                        | **required**             | Project Vault Key (`ZENV_PROJECT_KEY`) — never sent to server |
| `projectId`   | `string`                        | **required**             | Project ID |
| `schema`      | Standard Schema / plain object  | **required**             | Declares which secrets are safe to inject into the browser bundle |
| `environment` | `"development" \| "staging" \| "production"` | `"development"` | Target environment |
| `prefix`      | `string`                        | `"ZENV_"`                | Prefix for `import.meta.env` keys. `""` = no prefix |
| `baseUrl`     | `string`                        | `"https://api.zenv.sh"` | Override for self-hosted deployments |
| `watch`       | `number \| false`               | `30`                     | Dev polling interval in seconds. `0` or `false` = disabled |

## Schema

The `schema` option is **required**. This is a deliberate security decision — without it, the plugin would have no way to distinguish client-safe secrets from server-only credentials like `DATABASE_URL`.

```ts
// Zod (recommended)
import { z } from "zod"

schema: z.object({
  STRIPE_PUBLISHABLE_KEY: z.string().startsWith("pk_"),
  API_BASE_URL:           z.string().url(),
})

// Valibot
import * as v from "valibot"

schema: v.object({
  STRIPE_PUBLISHABLE_KEY: v.string(),
  API_BASE_URL:           v.string(),
})

// ArkType
import { type } from "arktype"

schema: type({
  STRIPE_PUBLISHABLE_KEY: "string",
  API_BASE_URL:           "string",
})

// Plain object — no validation, keys only
schema: {
  STRIPE_PUBLISHABLE_KEY: {},
  API_BASE_URL:           {},
}
```

> **Note on transforms:** Schema transforms (e.g. `z.string().transform(Number)`) are intentionally ignored at build time. `import.meta.env` values are always strings in the browser. Apply numeric transforms in your app code instead.

## Custom prefix

```ts
// No prefix → import.meta.env.STRIPE_PUBLISHABLE_KEY
zenvPlugin({ ..., prefix: "" })

// Custom prefix → import.meta.env.APP_STRIPE_PUBLISHABLE_KEY
zenvPlugin({ ..., prefix: "APP_" })
```

When using a custom prefix, extend `ImportMetaEnv` in your project's `env.d.ts`:

```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly [key: `APP_${string}`]: string | undefined
}
```

## Dev server — automatic secret sync

In `vite dev`, the plugin polls zEnv every `watch` seconds (default: 30). When a secret changes, the dev server restarts automatically — no manual action needed.

```ts
// Poll every 60 seconds
zenvPlugin({ ..., watch: 60 })

// Disable polling (secrets only load on server start)
zenvPlugin({ ..., watch: false })
```

You'll see a log line when secrets reload:

```
[zenv] Secrets changed — restarting dev server…
```

## Build behaviour

In `vite build`, the plugin fails hard if secrets cannot be fetched. This prevents shipping a broken bundle with undefined secret values. The error message includes the underlying API error for debugging.

## TypeScript

The plugin ships an ambient declaration that makes `import.meta.env.ZENV_*` valid TypeScript without errors:

```ts
// src/vite-env.d.ts — already included by Vite scaffolding
/// <reference types="vite/client" />
/// <reference types="@zenv/vite-plugin" />
```

## vs. @zenv/sdk

| | `@zenv/sdk` | `@zenv/vite-plugin` |
|---|---|---|
| **Runs in** | Node / server / edge | Vite build host (Node) |
| **Secrets available** | At runtime (fetched on demand) | At build time (bundled as literals) |
| **Browser safe** | ❌ (throws if `window` detected) | ✅ (values only, not credentials) |
| **Schema** | Optional | Required |
| **Use for** | APIs, SSR, CLI, backend | React, Vue, Svelte, etc. |
