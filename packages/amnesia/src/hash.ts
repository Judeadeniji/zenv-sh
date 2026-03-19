/**
 * Hashing — mirrors Go amnesia/hash.go
 *
 * HMAC-SHA256 for name hashing, Argon2id for Auth Key hashing.
 */
import { argon2id } from "hash-wasm";
import { toBuffer } from "./util.ts";

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
    toBuffer(hmacKey),
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
 * Params match Go exactly: m=64MB, t=3, p=4, output=32 bytes.
 * Salt = SHA-256(authKey)[:16] — deterministic, high-entropy input.
 *
 * Must produce byte-identical output to Go's amnesia.HashAuthKey().
 */
export async function hashAuthKey(authKey: Uint8Array): Promise<Uint8Array> {
  // Match Go: salt = sha256(authKey)[:16]
  const fullHash = new Uint8Array(
    await crypto.subtle.digest("SHA-256", toBuffer(authKey)),
  );
  const salt = fullHash.slice(0, 32); // Go uses sha256(authKey)[:SaltSize] where SaltSize=32

  const hash = await argon2id({
    password: authKey,
    salt,
    parallelism: 4,
    iterations: 3,
    memorySize: 64 * 1024, // 64 MB in KiB
    hashLength: 32,
    outputType: "binary",
  });

  return new Uint8Array(hash);
}
