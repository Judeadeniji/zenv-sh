/**
 * @zenv/vite-plugin — option types.
 */

/**
 * Options for the zEnv Vite plugin.
 *
 * Credentials (`token`, `projectKey`) are consumed entirely in Node at build
 * time and are never included in the browser bundle.
 */
export interface ZEnvPluginOptions {
	/**
	 * Service token — authenticates with the API.
	 * Typically read from `process.env.ZENV_TOKEN`.
	 */
	token: string

	/**
	 * Project Vault Key — derives the encryption key locally.
	 * **Never sent to the server. Never included in the browser bundle.**
	 * Typically read from `process.env.ZENV_PROJECT_KEY`.
	 */
	projectKey: string

	/**
	 * Project ID — which project to fetch secrets from.
	 * Typically read from `process.env.ZENV_PROJECT_ID`.
	 */
	projectId: string

	/**
	 * Target environment. Defaults to `"development"`.
	 * Maps to how you named the environment in the zEnv dashboard.
	 */
	environment?: "development" | "staging" | "production"

	/**
	 * Schema — **required**.
	 *
	 * Defines exactly which secrets should be fetched and injected into the
	 * browser bundle. Accepts any Standard Schema compliant validator (Zod,
	 * Valibot, ArkType) or a plain key-manifest object.
	 *
	 * Making this required is a deliberate security decision: the plugin
	 * refuses to operate as a bulk-export mechanism. You must explicitly
	 * declare which secrets are safe for the client.
	 *
	 * @example
	 * ```ts
	 * import { z } from "zod"
	 * schema: z.object({
	 *   STRIPE_PUBLISHABLE_KEY: z.string().startsWith("pk_"),
	 *   NEXT_PUBLIC_API_URL: z.string().url(),
	 * })
	 * ```
	 */
	schema: Record<string, unknown>

	/**
	 * Prefix prepended to every key in `import.meta.env`.
	 *
	 * - `"ZENV_"` (default) → `import.meta.env.ZENV_STRIPE_PUBLISHABLE_KEY`
	 * - `""` → `import.meta.env.STRIPE_PUBLISHABLE_KEY`
	 * - `"APP_"` → `import.meta.env.APP_STRIPE_PUBLISHABLE_KEY`
	 *
	 * The prefix does **not** need to match `VITE_` — the plugin uses
	 * Vite's `define` API which bypasses the `VITE_` restriction.
	 */
	prefix?: string

	/**
	 * API base URL. Defaults to `"https://api.zenv.sh"`.
	 * Override for self-hosted deployments.
	 */
	baseUrl?: string

	/**
	 * Dev-server polling interval (seconds).
	 *
	 * When running `vite dev`, the plugin polls zEnv on this interval and
	 * automatically restarts the dev server when a secret value changes.
	 * This means your app always reflects the current vault state without
	 * any manual action.
	 *
	 * - `30` (default) — poll every 30 seconds
	 * - `0` or `false` — disable polling (secrets only loaded on server start)
	 *
	 * Polling is disabled automatically during `vite build`.
	 */
	watch?: number | false
}

/** Resolved options with all defaults applied. */
export interface ResolvedZEnvPluginOptions
	extends Required<Omit<ZEnvPluginOptions, "watch">> {
	watch: number
}
