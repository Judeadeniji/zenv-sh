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
 *   });
 *   const secrets = await vault.load({ STRIPE_API_KEY: {}, DATABASE_URL: {} });
 */
import {
  deriveKeys,
  encrypt,
  decrypt,
  hashName,
  generateSalt,
} from "@zenv/amnesia";
import { createApiClient, type ApiClient } from "./client.ts";

export interface ZEnvConfig {
  /** Service token — authenticates with the API. */
  token: string;
  /** Project Vault Key — derives the encryption key locally. Never sent to server. */
  vaultKey: string;
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

    this.client = createApiClient({
      baseUrl: config.baseUrl ?? "https://api.zenv.sh",
      token: config.token,
    });
    this.vaultKey = config.vaultKey;
  }

  /**
   * Initialize crypto state — derives Project KEK from ZENV_VAULT_KEY,
   * unwraps the Project DEK. Called automatically on first operation.
   */
  private async initCrypto(): Promise<CryptoState> {
    if (this.crypto) return this.crypto;

    // TODO: fetch project salt + wrapped DEK from API once project crypto endpoints exist.
    // For now, derive a deterministic key from the vault key directly.
    const salt = generateSalt();
    const { kek } = await deriveKeys(this.vaultKey, salt, "passphrase");

    this.crypto = {
      dek: kek,
      hmacKey: kek,
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
        project_id: "", // TODO
        environment: "", // TODO
      } as any,
    });

    if (error) {
      throw new Error(`[zEnv] Failed to fetch secrets: ${JSON.stringify(error)}`);
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
        errors.push(`  ${name} — secret not found`);
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
        query: { project_id: "", environment: "" }, // TODO
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

    // Try update first, create if 404
    const { error } = await this.client.PUT("/sdk/secrets/{nameHash}", {
      params: {
        path: { nameHash: hash },
        query: { project_id: "", environment: "" }, // TODO
      },
      body: {
        ciphertext: bytesToBase64(ciphertext),
        nonce: bytesToBase64(nonce),
      },
    });

    if (error) {
      // Create new secret
      await this.client.POST("/sdk/secrets", {
        body: {
          name_hash: hash,
          ciphertext: bytesToBase64(ciphertext),
          nonce: bytesToBase64(nonce),
          project_id: "", // TODO
          environment: "", // TODO
        },
      });
    }
  }

  /** Delete a secret. */
  async delete(name: string): Promise<void> {
    const { dek, hmacKey } = await this.initCrypto();
    const hash = bytesToHex(await hashName(name, hmacKey));

    await this.client.DELETE("/sdk/secrets/{nameHash}", {
      params: {
        path: { nameHash: hash },
        query: { project_id: "", environment: "" }, // TODO
      },
    });
  }
}
