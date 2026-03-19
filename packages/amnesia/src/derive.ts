/**
 * Key derivation — mirrors Go amnesia/derive.go
 *
 * Single Argon2id run, 64-byte output split:
 *   bytes 0-31 → KEK
 *   bytes 32-63 → Auth Key
 */
import { argon2id } from "hash-wasm";

export type KeyType = "pin" | "passphrase";

interface Argon2Params {
  memorySize: number; // KiB
  iterations: number;
  parallelism: number;
}

const PIN_PARAMS: Argon2Params = {
  memorySize: 256 * 1024, // 256 MB in KiB
  iterations: 10,
  parallelism: 4,
};

const PASSPHRASE_PARAMS: Argon2Params = {
  memorySize: 64 * 1024, // 64 MB in KiB
  iterations: 3,
  parallelism: 4,
};

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
    hashLength: 64,
    outputType: "binary",
  });

  const output = new Uint8Array(hash);
  return {
    kek: output.slice(0, 32),
    authKey: output.slice(32, 64),
  };
}
