/**
 * @zenv/sdk — ZEnv class and `zenv()` factory.
 *
 * Zero crypto logic here. `@zenv/amnesia` handles everything cryptographic.
 * This module owns: API calls, schema validation, typed returns.
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

import { decrypt, deriveKeys, encrypt, hashName, unwrapKey } from "@zenv/amnesia"
import { type ApiClient, createApiClient } from "./client.ts"
import { base64ToBytes, bytesToBase64, bytesToHex } from "./encoding.ts"
import {
	ZEnvBrowserError,
	ZEnvConfigError,
	ZEnvFetchError,
	ZEnvNotFoundError,
	ZEnvStrictModeError,
	ZEnvValidationError,
} from "./errors.ts"
import { extractKeys, type InferSchema, pickSchema, validateValues } from "./schema.ts"
import type { CryptoState, ZEnvConfig } from "./types.ts"

const textEncoder = new TextEncoder()
const textDecoder = new TextDecoder()

export class ZEnv<S extends Record<string, unknown> = Record<string, unknown>> {
	private readonly client: ApiClient
	private readonly projectKey: string
	private readonly projectId: string
	private readonly environment: "development" | "staging" | "production"
	private readonly schema: S | undefined
	private readonly strict: boolean
	private readonly disableValidation: boolean
	private crypto: CryptoState | null = null

	constructor(config: ZEnvConfig<S>) {
		// Browser ban — credentials must never reach the browser.
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		if (typeof globalThis.window !== "undefined") {
			throw new ZEnvBrowserError()
		}

		if (!config.token) {
			throw new ZEnvConfigError(
				"[zEnv] Missing ZENV_TOKEN. Set it in your environment:\n" + "  export ZENV_TOKEN=ze_...",
			)
		}
		if (!config.projectKey) {
			throw new ZEnvConfigError(
				"[zEnv] Missing ZENV_PROJECT_KEY. Set it in your environment:\n" +
					"  export ZENV_PROJECT_KEY=...",
			)
		}
		if (!config.projectId) {
			throw new ZEnvConfigError(
				"[zEnv] Missing projectId. Set it in your config or via ZENV_PROJECT_ID.",
			)
		}

		if (config.disableValidation) {
			console.warn(
				"[zEnv] disableValidation is enabled. Schema value validation will not run. Not recommended for production.",
			)
		}

		this.client = createApiClient({
			baseUrl: config.baseUrl ?? "https://api.zenv.sh",
			token: config.token,
		})
		this.projectKey = config.projectKey
		this.projectId = config.projectId
		this.environment = config.environment ?? "development"
		this.schema = config.schema
		this.strict = config.strict ?? true
		this.disableValidation = config.disableValidation ?? false
	}

	/**
	 * Initialise crypto state by fetching project crypto from the API.
	 * Lazy and cached — runs at most once per `ZEnv` instance lifetime.
	 *
	 * Flow:
	 * 1. `GET /sdk/projects/{id}/crypto` → `project_salt` + `wrapped_project_dek`
	 * 2. `Argon2id(ZENV_PROJECT_KEY, project_salt)` → Project KEK
	 * 3. `AES-256-GCM unwrap(wrapped_project_dek, Project KEK)` → Project DEK
	 * 4. Cache `{ dek: ProjectDEK, hmacKey: ProjectDEK }` for the session.
	 */
	private async initCrypto(): Promise<CryptoState> {
		if (this.crypto) return this.crypto

		const { data, error } = await this.client.GET("/sdk/projects/{projectID}/crypto", {
			params: { path: { projectID: this.projectId } },
		})

		if (error || !data) {
			throw new ZEnvFetchError(
				`[zEnv] Failed to fetch project crypto for project '${this.projectId}'. ` +
					"Is the project ID correct and does the service token have access?",
			)
		}

		const { project_salt, wrapped_project_dek } = data
		const projectSalt = base64ToBytes(project_salt!)
		const wrappedProjectDEK = base64ToBytes(wrapped_project_dek!)

		const { kek: projectKEK } = await deriveKeys(this.projectKey, projectSalt, "passphrase")

		const wrappedNonce = wrappedProjectDEK.slice(0, 12)
		const wrappedCiphertext = wrappedProjectDEK.slice(12)
		const projectDEK = await unwrapKey(wrappedCiphertext, wrappedNonce, projectKEK)

		this.crypto = { dek: projectDEK, hmacKey: projectDEK }
		return this.crypto
	}

	/**
	 * Load secrets.
	 *
	 * **With a schema** (constructor or argument): fetches only the declared
	 * keys, validates + transforms values, returns a typed object.
	 *
	 * **Without a schema**: fetches ALL secrets for the project/environment as
	 * raw strings. Use this for quick scripts or debugging — prefer a schema
	 * in application code.
	 *
	 * All validation errors are collected and thrown together so you can fix
	 * everything in one pass.
	 */
	async load(): Promise<InferSchema<S>>
	async load<O extends Record<string, unknown>>(schema: O): Promise<InferSchema<O>>
	async load<O extends Record<string, unknown>>(schema?: O): Promise<InferSchema<O>> {
		const activeSchema = (schema ?? this.schema) as O | undefined
		const { dek, hmacKey } = await this.initCrypto()

		let nameHashes: { name: string; hash: string }[]

		if (activeSchema) {
			const keys = extractKeys(activeSchema)
			if (keys.length === 0) {
				throw new ZEnvConfigError("[zEnv] Schema has no keys. Define the secrets your app needs.")
			}
			nameHashes = await Promise.all(
				keys.map(async (name) => ({
					name,
					hash: bytesToHex(await hashName(name, hmacKey)),
				})),
			)
		} else {
			// No schema — list all secrets, then bulk fetch.
			const { data: listData, error: listError } = await this.client.GET("/sdk/secrets", {
				params: {
					query: { project_id: this.projectId, environment: this.environment },
				},
			})

			if (listError || !listData) {
				throw new ZEnvFetchError(`[zEnv] Failed to list secrets: ${JSON.stringify(listError)}`)
			}

			const rows = listData.secrets ?? []
			nameHashes = rows.map((r) => ({ name: r.name_hash!, hash: r.name_hash! }))

			if (nameHashes.length === 0) return {} as InferSchema<O>
		}

		const decrypted = await this.#bulkFetchAndDecrypt(nameHashes, dek)

		// Validate + transform if schema present and validation not disabled.
		if (activeSchema && !this.disableValidation) {
			const { result, errors: validationErrors } = await validateValues(activeSchema, decrypted)
			if (validationErrors.length > 0) {
				throw new ZEnvValidationError(validationErrors)
			}
			return result as InferSchema<O>
		}

		return decrypted as InferSchema<O>
	}

	/**
	 * Fetch a subset of secrets in a single bulk request.
	 *
	 * Three calling styles:
	 * ```ts
	 * vault.select("KEY1", "KEY2")          // variadic strings
	 * vault.select(["KEY1", "KEY2"])         // string array
	 * vault.select(z.object({ KEY1: ... })) // schema (extracts keys + validates)
	 * ```
	 *
	 * In strict mode, string-based calls validate requested names against the
	 * constructor schema before making any network request.
	 */
	async select(schema: Record<string, unknown>): Promise<Record<string, unknown>>
	async select(names: string[]): Promise<Record<string, string>>
	async select(...names: string[]): Promise<Record<string, string>>
	async select(...args: unknown[]): Promise<Record<string, unknown>> {
		let names: string[]
		let validationSchema: Record<string, unknown> | undefined

		if (args.length === 1 && typeof args[0] === "object" && !Array.isArray(args[0])) {
			validationSchema = args[0] as Record<string, unknown>
			names = extractKeys(validationSchema)
		} else if (args.length === 1 && Array.isArray(args[0])) {
			names = args[0] as string[]
		} else {
			names = args as string[]
		}

		if (names.length === 0) {
			throw new ZEnvConfigError("[zEnv] select() requires at least one key.")
		}

		// Strict mode: reject names not in constructor schema.
		if (this.strict && this.schema && !validationSchema) {
			const schemaKeys = extractKeys(this.schema)
			for (const name of names) {
				if (!schemaKeys.includes(name)) {
					throw new ZEnvStrictModeError(name, schemaKeys)
				}
			}
		}

		const { dek, hmacKey } = await this.initCrypto()
		const nameHashes = await Promise.all(
			names.map(async (name) => ({
				name,
				hash: bytesToHex(await hashName(name, hmacKey)),
			})),
		)

		const { data, error } = await this.client.POST("/sdk/secrets/bulk", {
			body: {
				name_hashes: nameHashes.map((n) => n.hash),
				project_id: this.projectId,
				environment: this.environment,
			},
		})

		if (error) {
			throw new ZEnvFetchError(`[zEnv] Failed to fetch secrets: ${JSON.stringify(error)}`)
		}

		const rows = data.secrets ?? []
		const rowMap = new Map<string, (typeof rows)[number]>()
		for (const row of rows) rowMap.set(row.name_hash!, row)

		const decrypted: Record<string, string> = {}
		const missing: string[] = []

		for (const { name, hash } of nameHashes) {
			const row = rowMap.get(hash)
			if (!row) {
				missing.push(name)
				continue
			}
			const plaintext = await decrypt(
				base64ToBytes(row.ciphertext!),
				base64ToBytes(row.nonce!),
				dek,
			)
			decrypted[name] = JSON.parse(textDecoder.decode(plaintext)).value
		}

		if (missing.length > 0) {
			throw new ZEnvNotFoundError(missing.join(", "), this.environment)
		}

		const activeSchema =
			validationSchema ?? (this.schema ? pickSchema(this.schema, names) : undefined)

		if (activeSchema && !this.disableValidation) {
			const { result, errors: valErrors } = await validateValues(activeSchema, decrypted)
			if (valErrors.length > 0) {
				throw new ZEnvValidationError(valErrors)
			}
			return result
		}

		return decrypted
	}

	/**
	 * Fetch a single secret by name.
	 * In strict mode, throws `ZEnvStrictModeError` for keys not in the schema.
	 */
	async get(name: string): Promise<string> {
		if (this.strict && this.schema) {
			const schemaKeys = extractKeys(this.schema)
			if (!schemaKeys.includes(name)) {
				throw new ZEnvStrictModeError(name, schemaKeys)
			}
		}

		const { dek, hmacKey } = await this.initCrypto()
		const hash = bytesToHex(await hashName(name, hmacKey))

		const { data, error } = await this.client.GET("/sdk/secrets/{nameHash}", {
			params: {
				path: { nameHash: hash },
				query: { project_id: this.projectId, environment: this.environment },
			},
		})

		if (error) {
			throw new ZEnvNotFoundError(name, this.environment)
		}

		const plaintext = await decrypt(
			base64ToBytes(data.ciphertext!),
			base64ToBytes(data.nonce!),
			dek,
		)
		const rawValue: string = JSON.parse(textDecoder.decode(plaintext)).value

		if (this.schema && !this.disableValidation) {
			const subSchema = pickSchema(this.schema, [name])
			const { result, errors } = await validateValues(subSchema, { [name]: rawValue })
			if (errors.length > 0) {
				throw new ZEnvValidationError(errors)
			}
			return result[name] as string
		}

		return rawValue
	}

	/**
	 * Encrypt and store a secret. Creates if new, updates if exists.
	 *
	 * The plaintext is never sent to the server — only the ciphertext and nonce.
	 */
	async set(name: string, value: string): Promise<void> {
		const { dek, hmacKey } = await this.initCrypto()
		const hash = bytesToHex(await hashName(name, hmacKey))

		const itemJson = JSON.stringify({ name, value })
		const plaintext = textEncoder.encode(itemJson)
		const { ciphertext, nonce } = await encrypt(plaintext, dek)

		const { error } = await this.client.PUT("/sdk/secrets/{nameHash}", {
			params: {
				path: { nameHash: hash },
				query: { project_id: this.projectId, environment: this.environment },
			},
			body: {
				ciphertext: bytesToBase64(ciphertext),
				nonce: bytesToBase64(nonce),
			},
		})

		if (error) {
			// PUT returned an error (likely 404 — secret doesn't exist yet). Fall back to POST.
			await this.client.POST("/sdk/secrets", {
				body: {
					name_hash: hash,
					ciphertext: bytesToBase64(ciphertext),
					nonce: bytesToBase64(nonce),
					project_id: this.projectId,
					environment: this.environment,
				},
			})
		}
	}

	/** Delete a secret from the vault. */
	async delete(name: string): Promise<void> {
		const { hmacKey } = await this.initCrypto()
		const hash = bytesToHex(await hashName(name, hmacKey))

		await this.client.DELETE("/sdk/secrets/{nameHash}", {
			params: {
				path: { nameHash: hash },
				query: { project_id: this.projectId, environment: this.environment },
			},
		})
	}

	// ─── Private helpers ─────────────────────────────────────────────────────

	/**
	 * Bulk fetch ciphertext from the API and decrypt each row.
	 * Used by `load()`.
	 */
	async #bulkFetchAndDecrypt(
		nameHashes: { name: string; hash: string }[],
		dek: Uint8Array,
	): Promise<Record<string, string>> {
		const { data, error } = await this.client.POST("/sdk/secrets/bulk", {
			body: {
				name_hashes: nameHashes.map((n) => n.hash),
				project_id: this.projectId,
				environment: this.environment,
			},
		})

		if (error) {
			throw new ZEnvFetchError(`[zEnv] Failed to fetch secrets: ${JSON.stringify(error)}`)
		}

		const rows = data.secrets ?? []
		const rowMap = new Map<string, (typeof rows)[number]>()
		for (const row of rows) rowMap.set(row.name_hash!, row)

		const decrypted: Record<string, string> = {}
		const missing: string[] = []

		for (const { name, hash } of nameHashes) {
			const row = rowMap.get(hash)
			if (!row) {
				missing.push(`  ${name} — secret not found in '${this.environment}'`)
				continue
			}
			const plaintext = await decrypt(
				base64ToBytes(row.ciphertext!),
				base64ToBytes(row.nonce!),
				dek,
			)
			const item = JSON.parse(textDecoder.decode(plaintext))
			// When loading without schema we don't know the original name (only the hash).
			// The decrypted payload always contains the name so we use it here.
			decrypted[item.name ?? name] = item.value
		}

		if (missing.length > 0) {
			throw new ZEnvValidationError(
				missing.map((m) => ({ key: m.trim().split(" — ")[0] ?? m, message: "not found" })),
			)
		}

		return decrypted
	}
}

/**
 * Create a new `ZEnv` vault instance.
 * Convenience wrapper around `new ZEnv(config)`.
 *
 * @example
 * ```ts
 * const vault = zenv({
 *   token:      process.env.ZENV_TOKEN!,
 *   projectKey: process.env.ZENV_PROJECT_KEY!,
 *   projectId:  process.env.ZENV_PROJECT_ID!,
 *   schema: z.object({ STRIPE_API_KEY: z.string() }),
 * })
 * const secrets = await vault.load()
 * ```
 */
export function zenv<S extends Record<string, unknown>>(config: ZEnvConfig<S>): ZEnv<S> {
	return new ZEnv(config)
}
