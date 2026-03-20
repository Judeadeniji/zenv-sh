/**
 * Hashing — mirrors Go amnesia/hash.go
 *
 * HMAC-SHA256 for name hashing, Argon2id for Auth Key hashing.
 */
import { argon2id } from "hash-wasm";
import { toBuffer } from "./util.ts";
import {
  PASSPHRASE_PARAMS,
  AUTH_KEY_HASH_LENGTH,
  AUTH_KEY_SALT_SIZE,
} from "./constants.ts";

const HMAC_ALGO: HmacImportParams = { name: "HMAC", hash: "SHA-256" };
const SIGN_USAGE: KeyUsage[] = ["sign"];
const encoder = new TextEncoder();

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
    HMAC_ALGO,
    false,
    SIGN_USAGE,
  );

  const sig = await crypto.subtle.sign("HMAC", cryptoKey, encoder.encode(name));
  return new Uint8Array(sig);
}

/**
 * Hash the Auth Key using Argon2id before sending to server.
 * Params match Go exactly: m=64MB, t=3, p=4, output=32 bytes.
 * Salt = SHA-256(authKey)[:32] — deterministic, high-entropy input.
 *
 * Must produce byte-identical output to Go's amnesia.HashAuthKey().
 */
export async function hashAuthKey(authKey: Uint8Array): Promise<Uint8Array> {
  const fullHash = new Uint8Array(
    await crypto.subtle.digest("SHA-256", toBuffer(authKey)),
  );
  const salt = fullHash.subarray(0, AUTH_KEY_SALT_SIZE);

  const hash = await argon2id({
    password: authKey,
    salt,
    parallelism: PASSPHRASE_PARAMS.parallelism,
    iterations: PASSPHRASE_PARAMS.iterations,
    memorySize: PASSPHRASE_PARAMS.memorySize,
    hashLength: AUTH_KEY_HASH_LENGTH,
    outputType: "binary",
  });

  return hash;
}
