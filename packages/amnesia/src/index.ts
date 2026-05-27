/**
 * @zenv/amnesia — Pure TypeScript cryptographic engine.
 *
 * Mirrors the Go amnesia/ package exactly.
 * No network. No storage. No concept of users, projects, or secrets.
 * Takes bytes in, gives bytes out.
 *
 * Cross-language parity with Go enforced via shared test vectors in CI.
 */

// Asymmetric crypto (X25519)
export {
	generateKeypair,
	unwrapWithPrivateKey,
	wrapWithPublicKey,
} from "./asymmetric.ts"
export type { Argon2Params } from "./constants.ts"
// Constants
export {
	CURVE25519_KEY_SIZE,
	DERIVED_KEY_SIZE,
	KEY_SIZE,
	NACL_HEADER_SIZE,
	NACL_NONCE_SIZE,
	NONCE_SIZE,
	PASSPHRASE_PARAMS,
	PIN_PARAMS,
	SALT_SIZE,
} from "./constants.ts"
export type { KeyType } from "./derive.ts"
// Key derivation
export { deriveKeys } from "./derive.ts"

// Hashing
export { hashAuthKey, hashName } from "./hash.ts"
// Random generation
export { generateKey, generateNonce, generateSalt } from "./random.ts"
// Symmetric encryption (AES-256-GCM)
export { decrypt, encrypt, unwrapKey, wrapKey } from "./symmetric.ts"
