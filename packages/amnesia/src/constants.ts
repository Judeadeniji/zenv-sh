/**
 * Shared constants — mirrors Go amnesia/ constants exactly.
 *
 * Every magic number in the package should trace back to a named
 * constant here. Keeps TS in lock-step with the Go definitions.
 */

// Symmetric (AES-256-GCM)
export const KEY_SIZE = 32;
export const NONCE_SIZE = 12;
export const SALT_SIZE = 32;

// Key derivation (Argon2id)
export const DERIVED_KEY_SIZE = 64; // 32-byte KEK + 32-byte Auth Key

// Asymmetric (NaCl box — X25519 + XSalsa20-Poly1305)
export const NACL_NONCE_SIZE = 24;
export const CURVE25519_KEY_SIZE = 32;
export const NACL_HEADER_SIZE = NACL_NONCE_SIZE + CURVE25519_KEY_SIZE; // 56

// Argon2id parameter sets
export interface Argon2Params {
  memorySize: number; // KiB
  iterations: number;
  parallelism: number;
}

export const PIN_PARAMS: Argon2Params = {
  memorySize: 256 * 1024, // 256 MiB
  iterations: 10,
  parallelism: 4,
};

export const PASSPHRASE_PARAMS: Argon2Params = {
  memorySize: 64 * 1024, // 64 MiB
  iterations: 3,
  parallelism: 4,
};

// hashAuthKey uses PASSPHRASE_PARAMS with a 32-byte output
export const AUTH_KEY_HASH_LENGTH = KEY_SIZE;
export const AUTH_KEY_SALT_SIZE = SALT_SIZE;
