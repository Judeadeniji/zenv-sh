/**
 * @zenv/sdk — Zero-knowledge secret manager SDK.
 *
 * @example
 * ```ts
 * import { zenv } from "@zenv/sdk"
 * import { z } from "zod"
 *
 * const vault = zenv({
 *   token:      process.env.ZENV_TOKEN!,
 *   projectKey: process.env.ZENV_PROJECT_KEY!,
 *   projectId:  process.env.ZENV_PROJECT_ID!,
 *   schema: z.object({
 *     STRIPE_API_KEY: z.string().min(1),
 *     DATABASE_URL:   z.string().url(),
 *     PORT:           z.string().transform(Number),
 *   }),
 * })
 *
 * const secrets = await vault.load()
 * // secrets.STRIPE_API_KEY → string
 * // secrets.PORT           → number (transformed by schema)
 * ```
 */

// Core — vault class and factory
export { ZEnv, zenv } from "./zenv.ts"

// Types — all public interfaces
export type { ZEnvConfig, CryptoState } from "./types.ts"
export type { InferSchema } from "./schema.ts"

// API client — for advanced/custom usage
export { createApiClient } from "./client.ts"
export type { ClientConfig, ApiClient } from "./client.ts"

// Errors — typed error hierarchy for instanceof checks
export {
	ZEnvError,
	ZEnvConfigError,
	ZEnvBrowserError,
	ZEnvFetchError,
	ZEnvValidationError,
	ZEnvNotFoundError,
	ZEnvStrictModeError,
} from "./errors.ts"

// Encoding — useful for consumers doing raw crypto operations
export { bytesToHex, bytesToBase64, base64ToBytes } from "./encoding.ts"
