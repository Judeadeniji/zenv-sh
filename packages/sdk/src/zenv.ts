/**
 * @zenv/sdk — The developer-facing SDK for zEnv.
 *
 * Zero crypto logic here. Amnesia handles everything.
 * This class owns: API calls, schema validation, typed returns.
 *
 * Usage:
 *   const vault = new ZEnv({
 *     token: process.env.ZENV_TOKEN,
 *     vaultKey: process.env.ZENV_VAULT_KEY,
 *     projectId: process.env.ZENV_PROJECT_ID,
 *     environment: process.env.NODE_ENV,
 *   });
 *   const secrets = await vault.load({ STRIPE_API_KEY: {}, DATABASE_URL: {} });
 */
import {
  deriveKeys,
  encrypt,
  decrypt,
  unwrapKey,
  hashName,
} from "@zenv/amnesia";
import { createApiClient, type ApiClient } from "./client.ts";

export interface ZEnvConfig {
  /** Service token — authenticates with the API. */
  token: string;
  /** Project Vault Key — derives the encryption key locally. Never sent to server. */
  vaultKey: string;
  /** Project ID — which project to fetch secrets from. */
  projectId: string;
  /** Environment — development, staging, or production. */
  environment?: string;
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

export class ZEnv {
  private client: ApiClient;
  private vaultKey: string;
  private projectId: string;
  private environment: string;
  private crypto: CryptoState | null = null;

  constructor(config: ZEnvConfig) {
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
          "  export ZENV_TOKEN=svc_...",
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

    this.client = createApiClient({
      baseUrl: config.baseUrl ?? "https://api.zenv.sh",
      token: config.token,
    });
    this.vaultKey = config.vaultKey;
    this.projectId = config.projectId;
    this.environment = config.environment ?? "development";
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

    // 1. Fetch project salt + wrapped DEK from API
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

    const projectSalt = base64ToBytes(
      (data as { project_salt: string }).project_salt,
    );
    const wrappedProjectDEK = base64ToBytes(
      (data as { wrapped_project_dek: string }).wrapped_project_dek,
    );

    // 2. Derive Project KEK from ZENV_VAULT_KEY + project salt
    const { kek: projectKEK } = await deriveKeys(
      this.vaultKey,
      projectSalt,
      "passphrase",
    );

    // 3. Unwrap Project DEK
    // Wrapped DEK format: first 12 bytes = nonce, rest = ciphertext (AES-256-GCM)
    const wrappedNonce = wrappedProjectDEK.slice(0, 12);
    const wrappedCiphertext = wrappedProjectDEK.slice(12);
    const projectDEK = await unwrapKey(wrappedCiphertext, wrappedNonce, projectKEK);

    // 4. Cache — DEK used for encrypt/decrypt, also as HMAC key for name hashing
    this.crypto = {
      dek: projectDEK,
      hmacKey: projectDEK,
    };

    return this.crypto;
  }

  /**
   * Load all secrets defined in a schema.
   *
   * The schema keys are the secret names. The SDK:
   * 1. Hashes each name with HMAC-SHA256
   * 2. Bulk fetches ciphertext from the API
   * 3. Decrypts each locally with Amnesia
   * 4. Returns a typed object matching the schema
   */
  async load<T extends Record<string, unknown>>(
    schema: Record<string, unknown>,
  ): Promise<T> {
    const { dek, hmacKey } = await this.initCrypto();
    const keys = Object.keys(schema);

    if (keys.length === 0) {
      throw new Error(
        "[zEnv] Schema has no keys. Define the secrets your app needs:\n" +
          '  await vault.load({ STRIPE_API_KEY: {}, DATABASE_URL: {} })',
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
      } as any,
    });

    if (error) {
      throw new Error(
        `[zEnv] Failed to fetch secrets: ${JSON.stringify(error)}`,
      );
    }

    // Build lookup: hash → row
    const rows = (data as any[]) ?? [];
    const rowMap = new Map<string, any>();
    for (const row of rows) {
      rowMap.set(row.name_hash, row);
    }

    // Decrypt each
    const result: Record<string, string> = {};
    const errors: string[] = [];

    for (const { name, hash } of nameHashes) {
      const row = rowMap.get(hash);
      if (!row) {
        errors.push(`  ${name} — secret not found in '${this.environment}'`);
        continue;
      }

      const plaintext = await decrypt(
        base64ToBytes(row.ciphertext),
        base64ToBytes(row.nonce),
        dek,
      );

      const item = JSON.parse(new TextDecoder().decode(plaintext));
      result[name] = item.value;
    }

    if (errors.length > 0) {
      throw new Error(
        `[zEnv] Startup validation failed:\n${errors.join("\n")}\n\n` +
          "Application did not start. Fix the above and retry.",
      );
    }

    return result as T;
  }

  /** Fetch a single secret by name. */
  async get(name: string): Promise<string> {
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

    const row = data as any;
    const plaintext = await decrypt(
      base64ToBytes(row.ciphertext),
      base64ToBytes(row.nonce),
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

    // Try update first, create if not found
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
