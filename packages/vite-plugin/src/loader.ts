/**
 * @zenv/vite-plugin — secret loader.
 *
 * Responsible for:
 *   - Resolving and validating plugin options (applying defaults)
 *   - Asserting that required credentials are present (fails fast with clear errors)
 *   - Fetching and decrypting secrets via `@zenv/sdk` (runs in Node, never browser)
 *   - Building the Vite `define` map from a secrets snapshot
 *
 * All functions here are pure or side-effect-free (no Vite server references),
 * making them straightforward to unit-test in isolation.
 */

import { ZEnv } from "@zenv/sdk"
import type { ResolvedZEnvPluginOptions, ZEnvPluginOptions } from "./types.ts"

/**
 * Apply defaults to raw user-supplied options.
 * The resolved form is used throughout the plugin lifecycle.
 */
export function resolveOptions(raw: ZEnvPluginOptions): ResolvedZEnvPluginOptions {
	return {
		token: raw.token,
		projectKey: raw.projectKey,
		projectId: raw.projectId,
		environment: raw.environment ?? "development",
		schema: raw.schema,
		prefix: raw.prefix ?? "ZENV_",
		baseUrl: raw.baseUrl ?? "https://api.zenv.sh",
		// false or 0 → 0 (disabled); omitted → 30 seconds
		watch: raw.watch === false || raw.watch === 0 ? 0 : (raw.watch ?? 30),
	}
}

/**
 * Assert that all required credentials are present and that `schema` is provided.
 *
 * Called before any network request so the developer sees a clear, actionable
 * error message before Vite even attempts to start.
 *
 * @throws {Error} if any required field is missing.
 */
export function assertCredentials(opts: ResolvedZEnvPluginOptions): void {
	const missing: string[] = []

	if (!opts.token) missing.push("`token` (ZENV_TOKEN)")
	if (!opts.projectKey) missing.push("`projectKey` (ZENV_PROJECT_KEY)")
	if (!opts.projectId) missing.push("`projectId` (ZENV_PROJECT_ID)")

	if (missing.length > 0) {
		throw new Error(
			`[zenv] vite-plugin-zenv is missing required credentials:\n\n` +
				missing.map((m) => `  • ${m}`).join("\n") +
				`\n\nSet them in your environment and read them in vite.config.ts:\n\n` +
				`  zenvPlugin({\n` +
				`    token:      process.env.ZENV_TOKEN!,\n` +
				`    projectKey: process.env.ZENV_PROJECT_KEY!,\n` +
				`    projectId:  process.env.ZENV_PROJECT_ID!,\n` +
				`    schema:     z.object({ ... }),\n` +
				`  })\n`,
		)
	}

	// Schema is required at the TypeScript level, but guard here for plain-JS users.
	if (!opts.schema || typeof opts.schema !== "object") {
		throw new Error(
			`[zenv] \`schema\` is required in vite-plugin-zenv options.\n\n` +
				`The schema declares exactly which secrets are safe to inject into the\n` +
				`browser bundle — without it the plugin has no way to distinguish\n` +
				`client-safe values from server-only credentials like DATABASE_URL.\n\n` +
				`Example:\n\n` +
				`  import { z } from "zod"\n\n` +
				`  zenvPlugin({\n` +
				`    // ...credentials...\n` +
				`    schema: z.object({\n` +
				`      STRIPE_PUBLISHABLE_KEY: z.string().startsWith("pk_"),\n` +
				`      NEXT_PUBLIC_API_URL:    z.string().url(),\n` +
				`    }),\n` +
				`  })\n`,
		)
	}
}

/**
 * Fetch and decrypt secrets from zEnv via the SDK.
 *
 * Runs entirely in Node (Vite's plugin host). Credentials are consumed here
 * and never forwarded to the browser. All values are returned as raw strings
 * because `import.meta.env` only supports string values.
 *
 * @throws if the API request or decryption fails.
 */
export async function fetchSecrets(
	opts: ResolvedZEnvPluginOptions,
): Promise<Record<string, string>> {
	const vault = new ZEnv({
		token: opts.token,
		projectKey: opts.projectKey,
		projectId: opts.projectId,
		environment: opts.environment,
		schema: opts.schema,
		baseUrl: opts.baseUrl,
		// Skip value validation — we embed raw strings into import.meta.env.
		// Schema transforms (z.string().transform(Number)) make no sense here:
		// import.meta.env values are always strings in the browser.
		disableValidation: true,
	})

	const result = (await vault.load()) as Record<string, unknown>

	// Coerce every value to string and warn on non-string values.
	const strings: Record<string, string> = {}
	for (const [key, value] of Object.entries(result)) {
		if (typeof value !== "string") {
			// Lazy import to avoid circular dep issues — logger is in the same package.
			const { warn } = await import("./logger.ts")
			warn(
				`Secret "${key}" has a non-string value after schema transformation. ` +
					`import.meta.env only supports strings — the raw value will be used. ` +
					`Apply numeric or boolean transforms in your app code instead.`,
			)
		}
		strings[key] = String(value)
	}

	return strings
}

/**
 * Build a Vite `define` map from a resolved secrets snapshot.
 *
 * Each secret key becomes `import.meta.env.<prefix><KEY>`.
 * Values are `JSON.stringify`'d so they are valid JS string literals in the bundle.
 *
 * @example
 * buildDefineMap({ STRIPE_KEY: "pk_live_..." }, "ZENV_")
 * // → { "import.meta.env.ZENV_STRIPE_KEY": '"pk_live_..."' }
 */
export function buildDefineMap(
	secrets: Record<string, string>,
	prefix: string,
): Record<string, string> {
	const map: Record<string, string> = {}
	for (const [key, value] of Object.entries(secrets)) {
		map[`import.meta.env.${prefix}${key}`] = JSON.stringify(value)
	}
	return map
}
