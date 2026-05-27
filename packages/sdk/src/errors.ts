/**
 * @zenv/sdk — typed error hierarchy.
 *
 * All SDK errors extend `ZEnvError`, so consumers can catch the base class
 * while still distinguishing specific failure modes with `instanceof`.
 *
 * @example
 * ```ts
 * import { ZEnvValidationError, ZEnvFetchError } from "@zenv/sdk"
 *
 * try {
 *   const secrets = await vault.load()
 * } catch (err) {
 *   if (err instanceof ZEnvValidationError) {
 *     // Schema mismatch — surface failures to the user.
 *     for (const { key, message } of err.failures) {
 *       console.error(`  ${key}: ${message}`)
 *     }
 *   } else if (err instanceof ZEnvFetchError) {
 *     // Network or API issue — safe to retry.
 *     console.error("Could not reach zEnv API:", err.message)
 *   } else {
 *     throw err // re-throw unexpected errors
 *   }
 * }
 * ```
 */

/** Base class for all zEnv SDK errors. */
export class ZEnvError extends Error {
	constructor(message: string) {
		super(message)
		this.name = "ZEnvError"
		// Maintain correct prototype chain in transpiled environments.
		Object.setPrototypeOf(this, new.target.prototype)
	}
}

/**
 * Thrown when required credentials (`token`, `projectKey`, `projectId`) are
 * missing or when `schema` is absent where it is required.
 */
export class ZEnvConfigError extends ZEnvError {
	constructor(message: string) {
		super(message)
		this.name = "ZEnvConfigError"
	}
}

/**
 * Thrown when the SDK is instantiated in a browser environment.
 * `ZENV_TOKEN` and `ZENV_PROJECT_KEY` are server credentials and must never
 * reach the browser. Use `@zenv/vite-plugin` for build-time injection.
 */
export class ZEnvBrowserError extends ZEnvError {
	constructor() {
		super(
			"[zEnv] @zenv/sdk detected a browser environment (window is defined). " +
				"ZENV_TOKEN and ZENV_PROJECT_KEY are server credentials — they must never " +
				"reach the browser. Use @zenv/vite-plugin for build-time injection instead.",
		)
		this.name = "ZEnvBrowserError"
	}
}

/** Thrown when an API request fails (network error, non-2xx response, etc.). */
export class ZEnvFetchError extends ZEnvError {
	constructor(message: string) {
		super(message)
		this.name = "ZEnvFetchError"
	}
}

/**
 * Thrown when schema validation fails on one or more secrets.
 * Contains the full list of per-key failures so callers can surface them all.
 */
export class ZEnvValidationError extends ZEnvError {
	/** All per-key validation failures from the current load/get/select. */
	readonly failures: ReadonlyArray<{ readonly key: string; readonly message: string }>

	constructor(failures: Array<{ key: string; message: string }>) {
		const lines = failures.map((f) => `  ${f.key} — ${f.message}`).join("\n")
		super(
			`[zEnv] Startup validation failed:\n${lines}\n\n` +
				"Application did not start. Fix the above secrets and retry.",
		)
		this.name = "ZEnvValidationError"
		this.failures = failures
	}
}

/**
 * Thrown when a requested secret is not found in the target environment.
 * Check that the secret exists in the zEnv dashboard for the environment
 * specified in your `ZEnvConfig`.
 */
export class ZEnvNotFoundError extends ZEnvError {
	/** The secret key that was not found. */
	readonly key: string

	constructor(key: string, environment: string) {
		super(`[zEnv] Secret "${key}" not found in environment "${environment}".`)
		this.name = "ZEnvNotFoundError"
		this.key = key
	}
}

/**
 * Thrown in strict mode when a key is accessed that is not declared in the schema.
 * Add the key to your schema or set `strict: false` to allow arbitrary access.
 */
export class ZEnvStrictModeError extends ZEnvError {
	constructor(key: string, definedKeys: string[]) {
		super(
			`[zEnv] '${key}' is not defined in your schema.\n` +
				`  Defined keys: ${definedKeys.join(", ")}\n` +
				`  Either add '${key}' to your schema or check for a typo.`,
		)
		this.name = "ZEnvStrictModeError"
	}
}
