/**
 * Asymmetric crypto — mirrors Go amnesia/asymmetric.go
 *
 * X25519 key exchange + AES-256-GCM for encrypting payloads to a public key.
 * Uses @noble/curves for X25519 (same curve as Go's crypto/ecdh).
 */
import { x25519 } from "@noble/curves/ed25519";
import { encrypt, decrypt } from "./symmetric.ts";
import { generateKey } from "./random.ts";

/**
 * Generate an X25519 keypair.
 * Returns 32-byte public key and 32-byte private key.
 */
export function generateKeypair(): {
  publicKey: Uint8Array;
  privateKey: Uint8Array;
} {
  const privateKey = generateKey(); // 32 random bytes
  const publicKey = x25519.getPublicKey(privateKey);
  return { publicKey, privateKey };
}

/**
 * Encrypt a payload for a recipient's public key.
 *
 * Protocol (matches Go implementation):
 * 1. Generate ephemeral X25519 keypair
 * 2. ECDH: shared secret = X25519(ephemeral private, recipient public)
 * 3. Derive AES key from shared secret via SHA-256
 * 4. AES-256-GCM encrypt payload with derived key
 * 5. Return: ephemeral public key (32) + nonce (12) + ciphertext
 */
export async function wrapWithPublicKey(
  payload: Uint8Array,
  recipientPublicKey: Uint8Array,
): Promise<Uint8Array> {
  // Ephemeral keypair
  const ephemeralPrivate = generateKey();
  const ephemeralPublic = x25519.getPublicKey(ephemeralPrivate);

  // ECDH shared secret
  const sharedSecret = x25519.getSharedSecret(
    ephemeralPrivate,
    recipientPublicKey,
  );

  // Derive AES key via SHA-256
  const aesKey = new Uint8Array(
    await crypto.subtle.digest("SHA-256", sharedSecret),
  );

  // Encrypt
  const { ciphertext, nonce } = await encrypt(payload, aesKey);

  // Pack: ephemeralPublic (32) + nonce (12) + ciphertext
  const out = new Uint8Array(32 + 12 + ciphertext.length);
  out.set(ephemeralPublic, 0);
  out.set(nonce, 32);
  out.set(ciphertext, 44);
  return out;
}

/**
 * Decrypt a payload encrypted with wrapWithPublicKey.
 *
 * Unpacks: ephemeral public key (32) + nonce (12) + ciphertext
 * Then reverses the ECDH + AES-256-GCM.
 */
export async function unwrapWithPrivateKey(
  packed: Uint8Array,
  recipientPrivateKey: Uint8Array,
): Promise<Uint8Array> {
  const ephemeralPublic = packed.slice(0, 32);
  const nonce = packed.slice(32, 44);
  const ciphertext = packed.slice(44);

  // ECDH shared secret
  const sharedSecret = x25519.getSharedSecret(
    recipientPrivateKey,
    ephemeralPublic,
  );

  // Derive AES key via SHA-256
  const aesKey = new Uint8Array(
    await crypto.subtle.digest("SHA-256", sharedSecret),
  );

  // Decrypt
  return decrypt(ciphertext, nonce, aesKey);
}
