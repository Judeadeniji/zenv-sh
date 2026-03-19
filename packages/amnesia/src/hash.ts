/**
 * Hashing — mirrors Go amnesia/hash.go
 *
 * HMAC-SHA256 for name hashing, Argon2id for Auth Key hashing.
 */
import { argon2id } from "hash-wasm";

/**
 * HMAC-SHA256 hash of a secret name for server-side indexed lookup.
 * Must produce byte-identical output to Go's amnesia.HashName().
 */
export async function hashName(
  name: string,
  hmacKey: Uint8Array,
): Promise<Uint8Array> {
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    hmacKey,
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );

  const data = new TextEncoder().encode(name);
  const sig = await crypto.subtle.sign("HMAC", cryptoKey, data);
  return new Uint8Array(sig);
}

/**
 * Hash the Auth Key using Argon2id before sending to server.
 * Uses fixed params matching Go: m=64MB, t=1, p=4, output=32 bytes.
 * Salt is the first 16 bytes of the Auth Key itself (deterministic).
 *
 * Must produce byte-identical output to Go's amnesia.HashAuthKey().
 */
export async function hashAuthKey(authKey: Uint8Array): Promise<Uint8Array> {
  const salt = authKey.slice(0, 16);

  const hash = await argon2id({
    password: authKey,
    salt,
    parallelism: 4,
    iterations: 1,
    memorySize: 64 * 1024, // 64 MB in KiB
    hashLength: 32,
    outputType: "binary",
  });

  return new Uint8Array(hash);
}
