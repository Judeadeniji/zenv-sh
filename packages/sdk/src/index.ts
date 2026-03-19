/**
 * @zenv/sdk — Zero-knowledge secret manager SDK.
 *
 * Usage:
 *   import { zenv } from "@zenv/sdk";
 *
 *   const vault = zenv({
 *     token: process.env.ZENV_TOKEN!,
 *     vaultKey: process.env.ZENV_VAULT_KEY!,
 *     projectId: process.env.ZENV_PROJECT_ID!,
 *     schema: z.object({
 *       STRIPE_API_KEY: z.string().min(1),
 *       DATABASE_URL: z.string().url(),
 *       PORT: z.string().transform(Number),
 *     }),
 *   });
 *
 *   const secrets = await vault.load();
 *   // secrets.STRIPE_API_KEY → string
 *   // secrets.PORT → number
 */

export { ZEnv, zenv } from "./zenv.ts";
export type { ZEnvConfig } from "./zenv.ts";
export { createApiClient } from "./client.ts";
export type { ClientConfig, ApiClient } from "./client.ts";
export type { InferSchema } from "./schema.ts";
