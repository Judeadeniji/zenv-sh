/**
 * @zenv-sh/sdk — shared type definitions.
 *
 * All public interfaces and internal types live here so contributors
 * can find them without navigating a large class file.
 */

/**
 * Configuration for a zEnv vault instance.
 *
 * @example
 * ```ts
 * const vault = zenv({
 *   token:      process.env.ZENV_TOKEN!,
 *   projectKey: process.env.ZENV_PROJECT_KEY!,
 *   projectId:  process.env.ZENV_PROJECT_ID!,
 *   schema: z.object({ STRIPE_API_KEY: z.string() }),
 * })
 * ```
 */
export interface ZEnvConfig<S extends Record<string, unknown> = Record<string, unknown>> {
	/** Service token — authenticates with the API. */
	token: string
	/**
	 * Project Vault Key — derives the encryption key locally.
	 * **Never sent to the server.**
	 */
	projectKey: string
	/** Project ID — which project to fetch secrets from. */
	projectId: string
	/** Target environment. Defaults to `"development"`. */
	environment?: "development" | "staging" | "production"
	/**
	 * Schema — defines which secrets to fetch and how to validate them.
	 * Accepts any Standard Schema compliant validator (Zod, Valibot, ArkType)
	 * or a plain key-manifest object.
	 */
	schema?: S
	/**
	 * Strict mode (default: `true`).
	 * When `true`, `get()` and `select()` reject keys not in the schema.
	 */
	strict?: boolean
	/**
	 * Disable schema value validation (default: `false`).
	 * When `true`, the schema is used as a fetch manifest only — values
	 * pass through as raw strings without validation or transformation.
	 */
	disableValidation?: boolean
	/** API base URL. Defaults to `"https://api.zenv.sh"`. */
	baseUrl?: string
}

/**
 * Internal crypto state, lazily initialised and cached per `ZEnv` instance.
 * Produced by `initCrypto()` after unwrapping the project DEK.
 *
 * @internal
 */
export interface CryptoState {
	/** Project DEK — used for AES-256-GCM encryption/decryption. */
	dek: Uint8Array
	/**
	 * HMAC key — used for HMAC-SHA256 name hashing.
	 * Currently the same bytes as `dek` (one DEK, dual role).
	 */
	hmacKey: Uint8Array
}
