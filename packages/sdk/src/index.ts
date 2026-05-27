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

export type { ApiClient, ClientConfig } from "./client.ts"
// API client — for advanced/custom usage
export { createApiClient } from "./client.ts"
// Encoding — useful for consumers doing raw crypto operations
export { base64ToBytes, bytesToBase64, bytesToHex } from "./encoding.ts"
// Errors — typed error hierarchy for instanceof checks
export {
	ZEnvBrowserError,
	ZEnvConfigError,
	ZEnvError,
	ZEnvFetchError,
	ZEnvNotFoundError,
	ZEnvStrictModeError,
	ZEnvValidationError,
} from "./errors.ts"
export type { InferSchema } from "./schema.ts"
// Types — all public interfaces
export type { CryptoState, ZEnvConfig } from "./types.ts"
// Core — vault class and factory
export { ZEnv, zenv } from "./zenv.ts"
