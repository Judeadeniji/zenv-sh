/**
 * Asymmetric crypto — mirrors Go amnesia/asymmetric.go
 *
 * Uses NaCl box (X25519 + XSalsa20-Poly1305) via tweetnacl.
 * This matches Go's golang.org/x/crypto/nacl/box exactly.
 *
 * Wire format: [24-byte nonce][32-byte ephemeral public key][sealed box]
 */
import nacl from "tweetnacl";

/**
 * Generate an X25519 keypair (NaCl box keypair).
 * Returns 32-byte public key and 32-byte private key.
 */
export function generateKeypair(): {
  publicKey: Uint8Array;
  privateKey: Uint8Array;
} {
  const kp = nacl.box.keyPair();
  return {
    publicKey: kp.publicKey,
    privateKey: kp.secretKey,
  };
}

/**
 * Encrypt a payload for a recipient's public key using NaCl box.
 *
 * Protocol (matches Go amnesia/asymmetric.go exactly):
 * 1. Generate ephemeral NaCl box keypair
 * 2. Generate random 24-byte nonce
 * 3. NaCl box.Seal(payload, nonce, recipientPub, ephemeralPriv)
 * 4. Return: nonce (24) + ephemeral public key (32) + sealed box
 */
export function wrapWithPublicKey(
  payload: Uint8Array,
  recipientPublicKey: Uint8Array,
): Uint8Array {
  // Ephemeral keypair
  const ephemeral = nacl.box.keyPair();

  // Random 24-byte nonce
  const nonce = nacl.randomBytes(24);

  // Seal
  const sealed = nacl.box(payload, nonce, recipientPublicKey, ephemeral.secretKey);

  // Pack: nonce (24) + ephemeral public key (32) + sealed box
  const out = new Uint8Array(24 + 32 + sealed.length);
  out.set(nonce, 0);
  out.set(ephemeral.publicKey, 24);
  out.set(sealed, 56);
  return out;
}

/**
 * Decrypt a payload encrypted with wrapWithPublicKey.
 *
 * Unpacks: nonce (24) + ephemeral public key (32) + sealed box
 * Then NaCl box.Open reverses the seal.
 */
export function unwrapWithPrivateKey(
  packed: Uint8Array,
  recipientPrivateKey: Uint8Array,
): Uint8Array {
  const minSize = 24 + 32 + nacl.box.overheadLength;
  if (packed.length < minSize) {
    throw new Error("amnesia: asymmetric decryption failed");
  }

  const nonce = packed.slice(0, 24);
  const ephemeralPublic = packed.slice(24, 56);
  const sealed = packed.slice(56);

  const plaintext = nacl.box.open(sealed, nonce, ephemeralPublic, recipientPrivateKey);
  if (plaintext === null) {
    throw new Error("amnesia: asymmetric decryption failed");
  }

  return plaintext;
}
