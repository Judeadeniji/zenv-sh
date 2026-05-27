import { describe, test } from "vitest";
import assert from "node:assert/strict";

import {
  generateSalt,
  generateNonce,
  generateKey,
  encrypt,
  decrypt,
  wrapKey,
  unwrapKey,
  hashName,
  hashAuthKey,
  deriveKeys,
  generateKeypair,
  wrapWithPublicKey,
  unwrapWithPrivateKey,
} from "./index.ts";

describe("random", () => {
  test("generateSalt produces 32 bytes", () => {
    const salt = generateSalt();
    assert.ok(salt instanceof Uint8Array);
    assert.strictEqual(salt.length, 32);
  });

  test("generateNonce produces 12 bytes", () => {
    const nonce = generateNonce();
    assert.ok(nonce instanceof Uint8Array);
    assert.strictEqual(nonce.length, 12);
  });

  test("generateKey produces 32 bytes", () => {
    const key = generateKey();
    assert.ok(key instanceof Uint8Array);
    assert.strictEqual(key.length, 32);
  });

  test("consecutive calls produce different output", () => {
    const a = generateKey();
    const b = generateKey();
    assert.notDeepStrictEqual(a, b);
  });
});

describe("symmetric", () => {
  test("encrypt/decrypt round-trip", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("hello zenv zero-knowledge");

    const { ciphertext, nonce } = await encrypt(plaintext, key);
    assert.ok(ciphertext.length > plaintext.length); // GCM tag adds 16 bytes

    const decrypted = await decrypt(ciphertext, nonce, key);
    assert.strictEqual(
      new TextDecoder().decode(decrypted),
      "hello zenv zero-knowledge",
    );
  });

  test("wrong key fails decryption", async () => {
    const key1 = generateKey();
    const key2 = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext, nonce } = await encrypt(plaintext, key1);
    await assert.rejects(decrypt(ciphertext, nonce, key2));
  });

  test("wrong nonce fails decryption", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext } = await encrypt(plaintext, key);
    const wrongNonce = generateNonce();
    await assert.rejects(decrypt(ciphertext, wrongNonce, key));
  });

  test("tampered ciphertext fails decryption", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext, nonce } = await encrypt(plaintext, key);
    ciphertext[0]! ^= 0xff; // flip bits
    await assert.rejects(decrypt(ciphertext, nonce, key));
  });
});

describe("key wrapping", () => {
  test("wrapKey/unwrapKey round-trip", async () => {
    const dek = generateKey();
    const kek = generateKey();

    const { ciphertext, nonce } = await wrapKey(dek, kek);
    const unwrapped = await unwrapKey(ciphertext, nonce, kek);

    assert.deepStrictEqual(unwrapped, dek);
  });
});

describe("hashing", () => {
  test("hashName produces 32-byte HMAC", async () => {
    const hmacKey = generateKey();
    const hash = await hashName("DATABASE_URL", hmacKey);
    assert.strictEqual(hash.length, 32);
  });

  test("hashName is deterministic", async () => {
    const hmacKey = generateKey();
    const h1 = await hashName("DATABASE_URL", hmacKey);
    const h2 = await hashName("DATABASE_URL", hmacKey);
    assert.deepStrictEqual(h1, h2);
  });

  test("hashName differs for different names", async () => {
    const hmacKey = generateKey();
    const h1 = await hashName("DATABASE_URL", hmacKey);
    const h2 = await hashName("API_KEY", hmacKey);
    assert.notDeepStrictEqual(h1, h2);
  });

  test("hashName differs for different keys", async () => {
    const k1 = generateKey();
    const k2 = generateKey();
    const h1 = await hashName("SECRET", k1);
    const h2 = await hashName("SECRET", k2);
    assert.notDeepStrictEqual(h1, h2);
  });

  test("hashAuthKey produces 32 bytes", async () => {
    const authKey = generateKey();
    const hash = await hashAuthKey(authKey);
    assert.strictEqual(hash.length, 32);
  });

  test("hashAuthKey is deterministic", async () => {
    const authKey = generateKey();
    const h1 = await hashAuthKey(authKey);
    const h2 = await hashAuthKey(authKey);
    assert.deepStrictEqual(h1, h2);
  });
});

describe("deriveKeys", () => {
  test("produces 32-byte KEK and 32-byte Auth Key", async () => {
    const salt = generateSalt();
    const { kek, authKey } = await deriveKeys("test-passphrase", salt, "passphrase");
    assert.strictEqual(kek.length, 32);
    assert.strictEqual(authKey.length, 32);
  });

  test("KEK and Auth Key are different", async () => {
    const salt = generateSalt();
    const { kek, authKey } = await deriveKeys("test-pin", salt, "pin");
    assert.notDeepStrictEqual(kek, authKey);
  });

  test("deterministic — same inputs produce same outputs", async () => {
    const salt = generateSalt();
    const r1 = await deriveKeys("my-vault-key", salt, "passphrase");
    const r2 = await deriveKeys("my-vault-key", salt, "passphrase");
    assert.deepStrictEqual(r1.kek, r2.kek);
    assert.deepStrictEqual(r1.authKey, r2.authKey);
  });

  test("different vault keys produce different outputs", async () => {
    const salt = generateSalt();
    const r1 = await deriveKeys("key-one", salt, "passphrase");
    const r2 = await deriveKeys("key-two", salt, "passphrase");
    assert.notDeepStrictEqual(r1.kek, r2.kek);
  });
});

describe("asymmetric", () => {
  test("generateKeypair produces 32-byte keys", () => {
    const { publicKey, privateKey } = generateKeypair();
    assert.strictEqual(publicKey.length, 32);
    assert.strictEqual(privateKey.length, 32);
  });

  test("wrapWithPublicKey/unwrapWithPrivateKey round-trip", async () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = new TextEncoder().encode("shared-secret-for-team");

    const packed = wrapWithPublicKey(payload, publicKey);
    assert.ok((await packed).length > 56); // 24 nonce + 32 ephPub + sealed

    const decrypted = unwrapWithPrivateKey(await packed, privateKey);
    assert.strictEqual(new TextDecoder().decode(await decrypted), "shared-secret-for-team");
  });

  test("wrong private key fails", async () => {
    const alice = generateKeypair();
    const bob = generateKeypair();
    const payload = new TextEncoder().encode("for-alice-only");

    const packed = wrapWithPublicKey(payload, alice.publicKey);
    await assert.rejects(async () => unwrapWithPrivateKey(await packed, bob.privateKey));
  });
});