/**
 * @zenv/sdk — The developer-facing SDK for zEnv.
 *
 * Zero crypto logic here. Amnesia handles everything.
 * This class owns: API calls, schema validation, typed returns.
 *
 * Usage:
 *   const vault = zenv({
 *     token: process.env.ZENV_TOKEN,
 *     vaultKey: process.env.ZENV_VAULT_KEY,
 *     projectId: process.env.ZENV_PROJECT_ID,
 *     environment: process.env.NODE_ENV,
 *     schema: z.object({
 *       STRIPE_API_KEY: z.string().min(1),
 *       DATABASE_URL: z.string().url(),
 *       PORT: z.string().transform(Number),
 *     }),
 *   });
 *   const secrets = await vault.load();
 */
import {
  deriveKeys,
  encrypt,
  decrypt,
  unwrapKey,
  hashName,
} from "@zenv/amnesia";
import { createApiClient, type ApiClient } from "./client.ts";
import { extractKeys, validateValues, type InferSchema } from "./schema.ts";

export interface ZEnvConfig<S extends Record<string, unknown> = Record<string, unknown>> {
  /** Service token — authenticates with the API. */
  token: string;
  /** Project Vault Key — derives the encryption key locally. Never sent to server. */
  vaultKey: string;
  /** Project ID — which project to fetch secrets from. */
  projectId: string;
  /** Environment — development, staging, or production. */
  environment?: string;
  /** Schema — defines which secrets to fetch and how to validate them. */
  schema?: S;
  /**
   * Strict mode (default: true).
   * When true with a schema, get() rejects keys not in the schema.
   */
  strict?: boolean;
  /**
   * Disable schema value validation (default: false).
   * When true, schema is used as fetch manifest only — values pass through as strings.
   */
  disableValidation?: boolean;
  /** API base URL. Defaults to https://api.zenv.sh */
  baseUrl?: string;
}

interface CryptoState {
  dek: Uint8Array;
  hmacKey: Uint8Array;
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function base64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

export class ZEnv<S extends Record<string, unknown> = Record<string, unknown>> {
  private client: ApiClient;
  private vaultKey: string;
  private projectId: string;
  private environment: string;
  private schema: S | undefined;
  private strict: boolean;
  private disableValidation: boolean;
  private crypto: CryptoState | null = null;

  constructor(config: ZEnvConfig<S>) {
    // Browser ban — ZENV_TOKEN and ZENV_VAULT_KEY must never reach the browser.
    if (typeof globalThis.window !== "undefined") {
      throw new Error(
        "[zEnv] @zenv/sdk detected a browser environment (window is defined). " +
          "ZENV_TOKEN and ZENV_VAULT_KEY are server credentials — they must never " +
          "reach the browser. Use @zenv/vite-plugin for build-time injection instead.",
      );
    }

    if (!config.token) {
      throw new Error(
        "[zEnv] Missing ZENV_TOKEN. Set it in your environment:\n" +
          "  export ZENV_TOKEN=ze_...",
      );
    }
    if (!config.vaultKey) {
      throw new Error(
        "[zEnv] Missing ZENV_VAULT_KEY. Set it in your environment:\n" +
          "  export ZENV_VAULT_KEY=...",
      );
    }
    if (!config.projectId) {
      throw new Error(
        "[zEnv] Missing projectId. Set it in your config or via ZENV_PROJECT_ID.",
      );
    }

    if (config.disableValidation) {
      console.warn(
        "[zEnv] disableValidation is enabled. Schema value validation will not run. Not recommended for production.",
      );
    }

    this.client = createApiClient({
      baseUrl: config.baseUrl ?? "https://api.zenv.sh",
      token: config.token,
    });
    this.vaultKey = config.vaultKey;
    this.projectId = config.projectId;
    this.environment = config.environment ?? "development";
    this.schema = config.schema;
    this.strict = config.strict ?? true;
    this.disableValidation = config.disableValidation ?? false;
  }

  /**
   * Initialize crypto state by fetching project crypto from the API.
   *
   * Flow (matches master plan Section 2.4.3):
   * 1. GET /sdk/projects/{id}/crypto → project_salt + wrapped_project_dek
   * 2. Argon2id(ZENV_VAULT_KEY + project_salt) → Project KEK
   * 3. AES-256-GCM unwrap(wrapped_project_dek, Project KEK) → Project DEK
   * 4. Project DEK used for all encrypt/decrypt + HMAC operations
   *
   * Called automatically on first operation. Cached for session lifetime.
   */
  private async initCrypto(): Promise<CryptoState> {
    if (this.crypto) return this.crypto;

    const { data, error } = await this.client.GET(
      "/sdk/projects/{projectID}/crypto",
      {
        params: { path: { projectID: this.projectId } },
      },
    );

    if (error || !data) {
      throw new Error(
        `[zEnv] Failed to fetch project crypto for project '${this.projectId}'. ` +
          "Is the project ID correct and does the service token have access?",
      );
    }

    const { project_salt, wrapped_project_dek } = data;
    const projectSalt = base64ToBytes(project_salt!);
    const wrappedProjectDEK = base64ToBytes(wrapped_project_dek!);

    const { kek: projectKEK } = await deriveKeys(
      this.vaultKey,
      projectSalt,
      "passphrase",
    );

    const wrappedNonce = wrappedProjectDEK.slice(0, 12);
    const wrappedCiphertext = wrappedProjectDEK.slice(12);
    const projectDEK = await unwrapKey(
      wrappedCiphertext,
      wrappedNonce,
      projectKEK,
    );

    this.crypto = { dek: projectDEK, hmacKey: projectDEK };
    return this.crypto;
  }

  /**
   * Load all secrets defined in the schema.
   *
   * Requires a schema — either passed in the constructor or as an argument.
   * load() without a schema throws with a helpful error.
   *
   * The SDK:
   * 1. Extracts key names from the schema (fetch manifest)
   * 2. Hashes each name with HMAC-SHA256
   * 3. Bulk fetches ciphertext from the API
   * 4. Decrypts each locally with Amnesia
   * 5. Validates + transforms values via schema (unless disableValidation)
   * 6. Reports all errors at once — not one-at-a-time
   */
  async load(): Promise<InferSchema<S>>;
  async load<O extends Record<string, unknown>>(schema: O): Promise<InferSchema<O>>;
  async load<O extends Record<string, unknown>>(schema?: O): Promise<InferSchema<O>> {
    const activeSchema = (schema ?? this.schema) as O | undefined;

    if (!activeSchema) {
      throw new Error(
        "[zEnv] load() requires a schema. Define it in the constructor or pass it to load():\n\n" +
          "  // In constructor:\n" +
          "  const vault = zenv({\n" +
          "    schema: z.object({ STRIPE_API_KEY: z.string() }),\n" +
          "    ...\n" +
          "  });\n" +
          "  const secrets = await vault.load();\n\n" +
          "  // Or inline:\n" +
          "  const secrets = await vault.load({ STRIPE_API_KEY: {} });\n\n" +
          "  // For single keys without a schema, use get():\n" +
          "  const key = await vault.get('STRIPE_API_KEY');",
      );
    }

    const { dek, hmacKey } = await this.initCrypto();
    const keys = extractKeys(activeSchema);

    if (keys.length === 0) {
      throw new Error(
        "[zEnv] Schema has no keys. Define the secrets your app needs.",
      );
    }

    // Hash all names
    const nameHashes = await Promise.all(
      keys.map(async (name) => ({
        name,
        hash: bytesToHex(await hashName(name, hmacKey)),
      })),
    );

    // Bulk fetch from API
    const { data, error } = await this.client.POST("/sdk/secrets/bulk", {
      body: {
        name_hashes: nameHashes.map((n) => n.hash),
        project_id: this.projectId,
        environment: this.environment,
      },
    });

    if (error) {
      throw new Error(
        `[zEnv] Failed to fetch secrets: ${JSON.stringify(error)}`,
      );
    }

    const rows = (data) ?? [];
    const rowMap = new Map<string, any>();
    for (const row of rows) {
      rowMap.set(row.name_hash!, row);
    }

    // Decrypt each
    const decrypted: Record<string, string> = {};
    const fetchErrors: string[] = [];

    for (const { name, hash } of nameHashes) {
      const row = rowMap.get(hash);
      if (!row) {
        fetchErrors.push(
          `  ${name} — secret not found in '${this.environment}'`,
        );
        continue;
      }

      const plaintext = await decrypt(
        base64ToBytes(row.ciphertext),
        base64ToBytes(row.nonce),
        dek,
      );

      const item = JSON.parse(new TextDecoder().decode(plaintext));
      decrypted[name] = item.value;
    }

    if (fetchErrors.length > 0) {
      throw new Error(
        `[zEnv] Startup validation failed:\n${fetchErrors.join("\n")}\n\n` +
          "Application did not start. Fix the above and retry.",
      );
    }

    // Validate + transform (unless disabled)
    if (this.disableValidation) {
      return decrypted as InferSchema<O>;
    }

    const { result, errors: validationErrors } = await validateValues(
      activeSchema,
      decrypted,
    );

    if (validationErrors.length > 0) {
      const lines = validationErrors.map((e) => `  ${e.key} — ${e.message}`);
      throw new Error(
        `[zEnv] Startup validation failed:\n${lines.join("\n")}\n\n` +
          "Application did not start. Fix the above and retry.",
      );
    }

    return result as InferSchema<O>;
  }

  /** Fetch a single secret by name. */
  async get(name: string): Promise<string> {
    // Strict mode: reject keys not in schema
    if (this.strict && this.schema) {
      const schemaKeys = extractKeys(this.schema);
      if (!schemaKeys.includes(name)) {
        const defined = schemaKeys.join(", ");
        throw new Error(
          `[zEnv] '${name}' is not defined in your schema.\n` +
            `  Defined keys: ${defined}\n` +
            `  Either add '${name}' to your schema or check for a typo.`,
        );
      }
    }

    const { dek, hmacKey } = await this.initCrypto();
    const hash = bytesToHex(await hashName(name, hmacKey));

    const { data, error } = await this.client.GET("/sdk/secrets/{nameHash}", {
      params: {
        path: { nameHash: hash },
        query: { project_id: this.projectId, environment: this.environment },
      },
    });

    if (error) {
      throw new Error(`[zEnv] Secret '${name}' not found`);
    }

    const row = data;
    const plaintext = await decrypt(
      base64ToBytes(row.ciphertext!),
      base64ToBytes(row.nonce!),
      dek,
    );

    const item = JSON.parse(new TextDecoder().decode(plaintext));
    return item.value;
  }

  /** Store a secret. Encrypts locally, sends ciphertext to API. */
  async set(name: string, value: string): Promise<void> {
    const { dek, hmacKey } = await this.initCrypto();
    const hash = bytesToHex(await hashName(name, hmacKey));

    const itemJson = JSON.stringify({ name, value });
    const plaintext = new TextEncoder().encode(itemJson);
    const { ciphertext, nonce } = await encrypt(plaintext, dek);

    const { error } = await this.client.PUT("/sdk/secrets/{nameHash}", {
      params: {
        path: { nameHash: hash },
        query: { project_id: this.projectId, environment: this.environment },
      },
      body: {
        ciphertext: bytesToBase64(ciphertext),
        nonce: bytesToBase64(nonce),
      },
    });

    if (error) {
      await this.client.POST("/sdk/secrets", {
        body: {
          name_hash: hash,
          ciphertext: bytesToBase64(ciphertext),
          nonce: bytesToBase64(nonce),
          project_id: this.projectId,
          environment: this.environment,
        },
      });
    }
  }

  /** Delete a secret. */
  async delete(name: string): Promise<void> {
    const { hmacKey } = await this.initCrypto();
    const hash = bytesToHex(await hashName(name, hmacKey));

    await this.client.DELETE("/sdk/secrets/{nameHash}", {
      params: {
        path: { nameHash: hash },
        query: { project_id: this.projectId, environment: this.environment },
      },
    });
  }
}

/**
 * Create a new ZEnv vault instance. Convenience wrapper around `new ZEnv()`.
 *
 * @example
 * const vault = zenv({
 *   token: process.env.ZENV_TOKEN!,
 *   vaultKey: process.env.ZENV_VAULT_KEY!,
 *   projectId: process.env.ZENV_PROJECT_ID!,
 *   schema: z.object({
 *     STRIPE_API_KEY: z.string().min(1),
 *     DATABASE_URL: z.string().url(),
 *   }),
 * });
 * const secrets = await vault.load();
 */
export function zenv<S extends Record<string, unknown>>(
  config: ZEnvConfig<S>,
): ZEnv<S> {
  return new ZEnv(config);
}
