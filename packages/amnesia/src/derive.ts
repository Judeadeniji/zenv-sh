/**
 * Key derivation — mirrors Go amnesia/derive.go
 *
 * Single Argon2id run, 64-byte output split:
 *   bytes 0-31 → KEK
 *   bytes 32-63 → Auth Key
 */
import { argon2id } from "hash-wasm";
import {
  KEY_SIZE,
  DERIVED_KEY_SIZE,
  PIN_PARAMS,
  PASSPHRASE_PARAMS,
} from "./constants.ts";

export type KeyType = "pin" | "passphrase";

/**
 * Derive KEK and Auth Key from a Vault Key + salt.
 * Argon2id runs once, produces 64 bytes, split by convention.
 *
 * Must produce byte-identical output to Go's amnesia.DeriveKeys().
 */
export async function deriveKeys(
  vaultKey: string,
  salt: Uint8Array,
  keyType: KeyType,
): Promise<{ kek: Uint8Array; authKey: Uint8Array }> {
  const params = keyType === "pin" ? PIN_PARAMS : PASSPHRASE_PARAMS;

  const hash = await argon2id({
    password: vaultKey,
    salt,
    parallelism: params.parallelism,
    iterations: params.iterations,
    memorySize: params.memorySize,
    hashLength: DERIVED_KEY_SIZE,
    outputType: "binary",
  });

  const output = new Uint8Array(hash);
  return {
    kek: output.slice(0, KEY_SIZE),
    authKey: output.slice(KEY_SIZE, DERIVED_KEY_SIZE),
  };
}
