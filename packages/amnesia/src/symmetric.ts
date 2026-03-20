/**
 * AES-256-GCM symmetric encryption — mirrors Go amnesia/symmetric.go
 *
 * All functions use Web Crypto API for AES-256-GCM.
 * Nonces are 12 bytes (96 bits), generated fresh per operation.
 */
import { generateNonce } from "./random.ts";
import { toBuffer } from "./util.ts";

const AES_GCM = "AES-GCM";
const AES_ALGO = { name: AES_GCM } as const;
const KEY_USAGES: KeyUsage[] = ["encrypt", "decrypt"];

async function importKey(key: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey("raw", toBuffer(key), AES_ALGO, false, KEY_USAGES);
}

/**
 * Encrypt plaintext with AES-256-GCM.
 * Returns ciphertext (includes GCM auth tag) and a fresh random nonce.
 */
export async function encrypt(
  plaintext: Uint8Array,
  key: Uint8Array,
): Promise<{ ciphertext: Uint8Array; nonce: Uint8Array }> {
  const nonce = generateNonce();
  const cryptoKey = await importKey(key);

  const encrypted = await crypto.subtle.encrypt(
    { name: AES_GCM, iv: toBuffer(nonce) },
    cryptoKey,
    toBuffer(plaintext),
  );

  return {
    ciphertext: new Uint8Array(encrypted),
    nonce,
  };
}

/**
 * Decrypt ciphertext with AES-256-GCM.
 * Ciphertext must include the GCM auth tag (appended by encrypt).
 */
export async function decrypt(
  ciphertext: Uint8Array,
  nonce: Uint8Array,
  key: Uint8Array,
): Promise<Uint8Array> {
  const cryptoKey = await importKey(key);

  const decrypted = await crypto.subtle.decrypt(
    { name: AES_GCM, iv: toBuffer(nonce) },
    cryptoKey,
    toBuffer(ciphertext),
  );

  return new Uint8Array(decrypted);
}

/**
 * Wrap a DEK with a KEK using AES-256-GCM.
 * Functionally identical to encrypt — semantic wrapper for clarity.
 */
export async function wrapKey(
  dek: Uint8Array,
  kek: Uint8Array,
): Promise<{ ciphertext: Uint8Array; nonce: Uint8Array }> {
  return encrypt(dek, kek);
}

/**
 * Unwrap a DEK with a KEK using AES-256-GCM.
 * Functionally identical to decrypt — semantic wrapper for clarity.
 */
export async function unwrapKey(
  ciphertext: Uint8Array,
  nonce: Uint8Array,
  kek: Uint8Array,
): Promise<Uint8Array> {
  return decrypt(ciphertext, nonce, kek);
}
