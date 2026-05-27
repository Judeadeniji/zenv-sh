/**
 * @zenv/vite-plugin
 *
 * Vite plugin that fetches encrypted secrets from zEnv at build time,
 * decrypts them in Node, and injects the plaintext values into
 * `import.meta.env` via Vite's `define` API.
 *
 * Credentials (`ZENV_TOKEN`, `ZENV_PROJECT_KEY`) never reach the browser.
 * Only the decrypted values you explicitly declare in `schema` are bundled.
 *
 * @example
 * ```ts
 * // vite.config.ts
 * import { defineConfig } from "vite"
 * import { z } from "zod"
 * import { zenvPlugin } from "@zenv/vite-plugin"
 *
 * export default defineConfig({
 *   plugins: [
 *     zenvPlugin({
 *       token:      process.env.ZENV_TOKEN!,
 *       projectKey: process.env.ZENV_PROJECT_KEY!,
 *       projectId:  process.env.ZENV_PROJECT_ID!,
 *       schema: z.object({
 *         STRIPE_PUBLISHABLE_KEY: z.string().startsWith("pk_"),
 *         API_BASE_URL:           z.string().url(),
 *       }),
 *     }),
 *   ],
 * })
 * ```
 *
 * In your app:
 * ```ts
 * const key = import.meta.env.ZENV_STRIPE_PUBLISHABLE_KEY // string
 * ```
 */

export { zenvPlugin } from "./plugin.ts"
export type { ZEnvPluginOptions, ResolvedZEnvPluginOptions } from "./types.ts"
