/**
 * Secure random generation — mirrors Go amnesia/random.go
 */

const SALT_SIZE = 32;
const NONCE_SIZE = 12;
const KEY_SIZE = 32;

/** Generate a 32-byte cryptographically random salt. */
export function generateSalt(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(SALT_SIZE));
}

/** Generate a 12-byte cryptographically random nonce (AES-256-GCM). */
export function generateNonce(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(NONCE_SIZE));
}

/** Generate a 32-byte cryptographically random key. */
export function generateKey(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(KEY_SIZE));
}
