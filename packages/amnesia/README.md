# @zenv/amnesia

Pure TypeScript cryptographic engine for zEnv. Byte-identical reimplementation of the [Go Amnesia](../../amnesia/) package using Web Crypto API, hash-wasm, and tweetnacl.

No network. No storage. Takes bytes in, gives bytes out.

## API

```typescript
// Key derivation — Argon2id, 64 bytes split: KEK (0-31) + Auth Key (32-63)
deriveKeys(vaultKey: string, salt: Uint8Array, keyType: "pin" | "passphrase")
  → Promise<{ kek: Uint8Array; authKey: Uint8Array }>

// Symmetric encryption (AES-256-GCM via Web Crypto API)
encrypt(plaintext: Uint8Array, key: Uint8Array)
  → Promise<{ ciphertext: Uint8Array; nonce: Uint8Array }>
decrypt(ciphertext: Uint8Array, nonce: Uint8Array, key: Uint8Array)
  → Promise<Uint8Array>

// Key wrapping
wrapKey(dek: Uint8Array, kek: Uint8Array) → Promise<{ ciphertext, nonce }>
unwrapKey(ciphertext: Uint8Array, nonce: Uint8Array, kek: Uint8Array) → Promise<Uint8Array>

// Name hashing (HMAC-SHA256 via Web Crypto API)
hashName(name: string, hmacKey: Uint8Array) → Promise<Uint8Array>

// Auth Key hashing (Argon2id via hash-wasm)
hashAuthKey(authKey: Uint8Array) → Promise<Uint8Array>

// Asymmetric encryption (X25519 + NaCl box via tweetnacl)
generateKeypair() → { publicKey: Uint8Array; privateKey: Uint8Array }
wrapWithPublicKey(payload: Uint8Array, publicKey: Uint8Array) → Uint8Array
unwrapWithPrivateKey(packed: Uint8Array, privateKey: Uint8Array) → Uint8Array

// Secure random generation (crypto.getRandomValues)
generateSalt() → Uint8Array   // 32 bytes
generateNonce() → Uint8Array  // 12 bytes
generateKey() → Uint8Array    // 32 bytes
```

## Usage

```typescript
import { deriveKeys, encrypt, decrypt, generateSalt, generateKey } from "@zenv/amnesia";

const salt = generateSalt();
const { kek } = await deriveKeys("my-passphrase", salt, "passphrase");

const dek = generateKey();
const plaintext = new TextEncoder().encode("secret-value");
const { ciphertext, nonce } = await encrypt(plaintext, dek);

const decrypted = await decrypt(ciphertext, nonce, dek);
// new TextDecoder().decode(decrypted) === "secret-value"
```

## Dependencies

| Package | Purpose |
| --- | --- |
| [hash-wasm](https://github.com/nicolo-ribaudo/hash-wasm) | Argon2id (key derivation + auth key hashing) |
| [tweetnacl](https://github.com/nicolo-ribaudo/tweetnacl-js) | NaCl box (X25519 + XSalsa20-Poly1305) — matches Go's `nacl/box` |
| [@noble/curves](https://github.com/nicolo-ribaudo/noble-curves) | Not currently used (kept for future Ed25519 if needed) |

## Cross-Language Parity

This package must produce **byte-identical output** to the Go Amnesia for the same inputs. Parity is enforced by shared test vectors:

1. Go generates `tests/vectors.json` with known inputs and expected outputs
2. TypeScript validates against the same JSON file
3. CI runs both — any drift fails the build

Covered operations: `DeriveKeys`, AES-256-GCM encrypt/decrypt, `HashName` (HMAC-SHA256), `HashAuthKey` (Argon2id).

## Testing

```bash
bun test
```

22 unit tests + 8 cross-language parity tests = 30 total.
