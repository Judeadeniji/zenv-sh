import { test, expect, describe } from "bun:test";
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
    expect(salt).toBeInstanceOf(Uint8Array);
    expect(salt.length).toBe(32);
  });

  test("generateNonce produces 12 bytes", () => {
    const nonce = generateNonce();
    expect(nonce).toBeInstanceOf(Uint8Array);
    expect(nonce.length).toBe(12);
  });

  test("generateKey produces 32 bytes", () => {
    const key = generateKey();
    expect(key).toBeInstanceOf(Uint8Array);
    expect(key.length).toBe(32);
  });

  test("consecutive calls produce different output", () => {
    const a = generateKey();
    const b = generateKey();
    expect(a).not.toEqual(b);
  });
});

describe("symmetric", () => {
  test("encrypt/decrypt round-trip", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("hello zenv zero-knowledge");

    const { ciphertext, nonce } = await encrypt(plaintext, key);
    expect(ciphertext.length).toBeGreaterThan(plaintext.length); // GCM tag adds 16 bytes

    const decrypted = await decrypt(ciphertext, nonce, key);
    expect(new TextDecoder().decode(decrypted)).toBe(
      "hello zenv zero-knowledge",
    );
  });

  test("wrong key fails decryption", async () => {
    const key1 = generateKey();
    const key2 = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext, nonce } = await encrypt(plaintext, key1);
    await expect(decrypt(ciphertext, nonce, key2)).rejects.toThrow();
  });

  test("wrong nonce fails decryption", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext } = await encrypt(plaintext, key);
    const wrongNonce = generateNonce();
    await expect(decrypt(ciphertext, wrongNonce, key)).rejects.toThrow();
  });

  test("tampered ciphertext fails decryption", async () => {
    const key = generateKey();
    const plaintext = new TextEncoder().encode("secret");

    const { ciphertext, nonce } = await encrypt(plaintext, key);
    ciphertext[0]! ^= 0xff; // flip bits
    await expect(decrypt(ciphertext, nonce, key)).rejects.toThrow();
  });
});

describe("key wrapping", () => {
  test("wrapKey/unwrapKey round-trip", async () => {
    const dek = generateKey();
    const kek = generateKey();

    const { ciphertext, nonce } = await wrapKey(dek, kek);
    const unwrapped = await unwrapKey(ciphertext, nonce, kek);

    expect(unwrapped).toEqual(dek);
  });
});

describe("hashing", () => {
  test("hashName produces 32-byte HMAC", async () => {
    const hmacKey = generateKey();
    const hash = await hashName("DATABASE_URL", hmacKey);
    expect(hash.length).toBe(32);
  });

  test("hashName is deterministic", async () => {
    const hmacKey = generateKey();
    const h1 = await hashName("DATABASE_URL", hmacKey);
    const h2 = await hashName("DATABASE_URL", hmacKey);
    expect(h1).toEqual(h2);
  });

  test("hashName differs for different names", async () => {
    const hmacKey = generateKey();
    const h1 = await hashName("DATABASE_URL", hmacKey);
    const h2 = await hashName("API_KEY", hmacKey);
    expect(h1).not.toEqual(h2);
  });

  test("hashName differs for different keys", async () => {
    const k1 = generateKey();
    const k2 = generateKey();
    const h1 = await hashName("SECRET", k1);
    const h2 = await hashName("SECRET", k2);
    expect(h1).not.toEqual(h2);
  });

  test("hashAuthKey produces 32 bytes", async () => {
    const authKey = generateKey();
    const hash = await hashAuthKey(authKey);
    expect(hash.length).toBe(32);
  });

  test("hashAuthKey is deterministic", async () => {
    const authKey = generateKey();
    const h1 = await hashAuthKey(authKey);
    const h2 = await hashAuthKey(authKey);
    expect(h1).toEqual(h2);
  });
});

describe("deriveKeys", () => {
  test("produces 32-byte KEK and 32-byte Auth Key", async () => {
    const salt = generateSalt();
    const { kek, authKey } = await deriveKeys("test-passphrase", salt, "passphrase");
    expect(kek.length).toBe(32);
    expect(authKey.length).toBe(32);
  });

  test("KEK and Auth Key are different", async () => {
    const salt = generateSalt();
    const { kek, authKey } = await deriveKeys("test-pin", salt, "pin");
    expect(kek).not.toEqual(authKey);
  });

  test("deterministic — same inputs produce same outputs", async () => {
    const salt = generateSalt();
    const r1 = await deriveKeys("my-vault-key", salt, "passphrase");
    const r2 = await deriveKeys("my-vault-key", salt, "passphrase");
    expect(r1.kek).toEqual(r2.kek);
    expect(r1.authKey).toEqual(r2.authKey);
  });

  test("different vault keys produce different outputs", async () => {
    const salt = generateSalt();
    const r1 = await deriveKeys("key-one", salt, "passphrase");
    const r2 = await deriveKeys("key-two", salt, "passphrase");
    expect(r1.kek).not.toEqual(r2.kek);
  });
});

describe("asymmetric", () => {
  test("generateKeypair produces 32-byte keys", () => {
    const { publicKey, privateKey } = generateKeypair();
    expect(publicKey.length).toBe(32);
    expect(privateKey.length).toBe(32);
  });

  test("wrapWithPublicKey/unwrapWithPrivateKey round-trip", async () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = new TextEncoder().encode("shared-secret-for-team");

    const packed = wrapWithPublicKey(payload, publicKey);
    expect((await packed).length).toBeGreaterThan(56); // 24 nonce + 32 ephPub + sealed

    const decrypted = unwrapWithPrivateKey(await packed, privateKey);
    expect(new TextDecoder().decode(await decrypted)).toBe("shared-secret-for-team");
  });

  test("wrong private key fails", () => {
    const alice = generateKeypair();
    const bob = generateKeypair();
    const payload = new TextEncoder().encode("for-alice-only");

    const packed = wrapWithPublicKey(payload, alice.publicKey);
    expect(async () => unwrapWithPrivateKey(await packed, bob.privateKey)).toThrow();
  });
});
