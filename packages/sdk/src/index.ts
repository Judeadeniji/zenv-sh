/**
 * @zenv/sdk — Zero-knowledge secret manager SDK.
 *
 * Usage:
 *   import { ZEnv } from "@zenv/sdk";
 *
 *   const vault = new ZEnv({
 *     token: process.env.ZENV_TOKEN!,
 *     vaultKey: process.env.ZENV_VAULT_KEY!,
 *   });
 *
 *   // With Zod:
 *   const secrets = await vault.load(z.object({
 *     STRIPE_API_KEY: z.string().min(1),
 *     DATABASE_URL: z.string().url(),
 *     PORT: z.string().transform(Number),
 *   }));
 *
 *   // Or plain object (no validation):
 *   const secrets = await vault.load({ STRIPE_API_KEY: {}, DATABASE_URL: {} });
 */

export { ZEnv } from "./zenv.ts";
export type { ZEnvConfig } from "./zenv.ts";
export { createApiClient } from "./client.ts";
export type { ClientConfig, ApiClient } from "./client.ts";
export type { InferSchema } from "./schema.ts";
