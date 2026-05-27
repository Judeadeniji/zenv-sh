/**
 * TypeScript ambient declarations for @zenv/vite-plugin.
 *
 * Augments Vite's `ImportMetaEnv` so that `import.meta.env.ZENV_*` keys are
 * recognised by the TypeScript compiler without requiring manual declaration.
 *
 * This file ships with the package and is automatically picked up by
 * `tsconfig.json` when you add `@zenv/vite-plugin` to your project.
 *
 * For per-project key-level typing (e.g. `import.meta.env.ZENV_STRIPE_KEY`
 * typed as `string` rather than the generic index), run the codegen target
 * in your project's build pipeline (see README).
 */

/// <reference types="vite/client" />

interface ImportMetaEnv {
	// Generic catch-all for any key injected via the default "ZENV_" prefix.
	// Users who configure a custom prefix should extend this interface in their
	// own `env.d.ts` using the same pattern.
	readonly [key: `ZENV_${string}`]: string | undefined
}

interface ImportMeta {
	readonly env: ImportMetaEnv
}
