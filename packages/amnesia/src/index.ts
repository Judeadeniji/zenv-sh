/**
 * @zenv/amnesia — Pure TypeScript cryptographic engine.
 *
 * Mirrors the Go amnesia/ package exactly.
 * No network. No storage. No concept of users, projects, or secrets.
 * Takes bytes in, gives bytes out.
 *
 * Cross-language parity with Go enforced via shared test vectors in CI.
 */

// Key derivation
export { deriveKeys } from "./derive.ts";
export type { KeyType } from "./derive.ts";

// Symmetric encryption (AES-256-GCM)
export { encrypt, decrypt, wrapKey, unwrapKey } from "./symmetric.ts";

// Hashing
export { hashName, hashAuthKey } from "./hash.ts";

// Asymmetric crypto (X25519)
export {
  generateKeypair,
  wrapWithPublicKey,
  unwrapWithPrivateKey,
} from "./asymmetric.ts";

// Random generation
export { generateSalt, generateNonce, generateKey } from "./random.ts";
