# @zenv/sdk

TypeScript SDK for zEnv. Fetches encrypted secrets from the API, decrypts them locally via [@zenv/amnesia](../amnesia/), validates with your schema, and returns typed results. Zero crypto logic in this package.

## Install

```bash
pnpm add @zenv/sdk
```

## Usage

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
// secrets.DATABASE_URL   → string
// secrets.PORT           → number (transformed by schema)
```

## Config

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `token` | `string` | required | Service token (`ZENV_TOKEN`) |
| `projectKey` | `string` | required | Project Project Key (`ZENV_PROJECT_KEY`) — never sent to server |
| `projectId` | `string` | required | Project ID |
| `environment` | `string` | `"development"` | Environment name |
| `schema` | `object` | optional | Standard Schema compliant validator (Zod, Valibot, ArkType) or plain object |
| `strict` | `boolean` | `true` | Reject `get()` calls for keys not in schema |
| `disableValidation` | `boolean` | `false` | Skip schema value validation (keys still used as fetch manifest) |
| `baseUrl` | `string` | `"https://api.zenv.sh"` | API base URL |

## Methods

### `vault.load()`

Fetch all secrets defined in the schema. Decrypts locally, validates, returns typed object. Reports all errors at once.

```typescript
const secrets = await vault.load();
```

### `vault.load(schema)`

Override the constructor schema for a one-off load.

```typescript
const secrets = await vault.load({ API_KEY: {}, DB_URL: {} });
```

### `vault.get(name)`

Fetch a single secret by name. In strict mode, rejects keys not in the schema.

```typescript
const apiKey = await vault.get("STRIPE_API_KEY");
```

### `vault.set(name, value)`

Encrypt and store a secret. Creates if new, updates if exists.

```typescript
await vault.set("DATABASE_URL", "postgres://prod:secret@db/myapp");
```

### `vault.delete(name)`

Delete a secret from the vault.

```typescript
await vault.delete("OLD_API_KEY");
```

## Schema Support

The SDK accepts any [Standard Schema](https://github.com/standard-schema/standard-schema) v1 compliant validator. The schema serves two purposes:

1. **Fetch manifest** — keys define which secrets to fetch
2. **Validation + transformation** — values are validated after decryption

```typescript
// Zod
schema: z.object({ PORT: z.string().transform(Number) })

// Valibot
schema: v.object({ PORT: v.string() })

// ArkType
schema: type({ PORT: "string" })

// Plain object (no validation, keys only)
schema: { PORT: {} }
```

## Behaviour Matrix

| Schema | Method | strict | disableValidation | Behaviour |
| --- | --- | --- | --- | --- |
| Defined | `load()` | true | false | Fetch schema keys, validate, return typed object |
| Defined | `get(valid)` | true | false | Fetch, validate against schema field |
| Defined | `get(invalid)` | true | false | Throw immediately — key not in schema |
| Defined | `get(any)` | false | false | Fetch as raw string, known keys still validated |
| Defined | any | any | true | Key enforcement per strict, value validation skipped |
| None | `load()` | - | - | Throw with helpful error and code examples |
| None | `get(any)` | - | - | Fetch and return raw string |

## Browser Ban

The SDK throws a hard error if `window` is defined. `ZENV_TOKEN` and `ZENV_PROJECT_KEY` are server credentials — they must never reach the browser.

For browser/edge runtimes, use `@zenv/vite-plugin` which moves all crypto to build time.

## How It Works

```
ZENV_PROJECT_KEY + project_salt (from API)
  → Argon2id → Project KEK (client-side, never sent)
    → AES-256-GCM unwrap → Project DEK (client-side, never sent)
      → AES-256-GCM decrypt → plaintext secrets
```

The server never sees the Project Key, KEK, or DEK. It stores only ciphertext.
