// Test Amnesia WASM in Node.js
// Usage: node tests/wasm_test.mjs

import { readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// Load TinyGo's wasm_exec.js glue
const glueCode = readFileSync(join(__dirname, "../wasm/wasm_exec.js"), "utf-8");
new Function(glueCode)();

const go = new Go();

// Load and instantiate the WASM module
const wasmPath = join(__dirname, "../wasm/amnesia.wasm");
const wasmBuffer = readFileSync(wasmPath);
const { instance } = await WebAssembly.instantiate(wasmBuffer, go.importObject);

// Start the Go runtime (non-blocking for TinyGo)
go.run(instance);

const exports = instance.exports;
const memory = exports.memory;

// --- Helpers ---

function readResult() {
  const ptr = exports.getResultPtr();
  const len = exports.getResultLen();
  return new Uint8Array(memory.buffer, ptr, len).slice();
}

function readResultString() {
  const buf = readResult();
  return new TextDecoder().decode(buf);
}

function writeBytes(data) {
  const ptr = exports.allocate(data.length);
  const view = new Uint8Array(memory.buffer, ptr, data.length);
  view.set(data);
  return [ptr, data.length];
}

function writeString(str) {
  return writeBytes(new TextEncoder().encode(str));
}

function isError(result) {
  return new TextDecoder().decode(result).startsWith("ERR:");
}

let pass = 0;
let fail = 0;

function assert(label, condition) {
  if (condition) {
    console.log(`  \x1b[32m✓\x1b[0m ${label}`);
    pass++;
  } else {
    console.log(`  \x1b[31m✗\x1b[0m ${label}`);
    fail++;
  }
}

function arraysEqual(a, b) {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

// --- Tests ---

console.log("=== Random Generation ===");

exports.generateSalt();
const salt = readResult();
assert("generateSalt returns 32 bytes", salt.length === 32);

exports.generateNonce();
const nonce = readResult();
assert("generateNonce returns 12 bytes", nonce.length === 12);

exports.generateKey();
const key = readResult();
assert("generateKey returns 32 bytes", key.length === 32);

exports.generateSalt();
const salt2 = readResult();
assert("two salts differ", !arraysEqual(salt, salt2));

console.log("\n=== Key Derivation ===");
console.log("  (skipped — Argon2id uses goroutines, not supported in TinyGo WASM)");
console.log("  (SDK uses JS-side argon2 lib, passes derived KEK to WASM for encrypt/decrypt)");

// Generate a key to use as KEK for remaining tests
exports.generateKey();
const kek = readResult();

console.log("\n=== Encrypt / Decrypt ===");

const plaintext = new TextEncoder().encode('{"name":"DB_URL","value":"postgres://localhost"}');
exports.generateKey();
const dek = readResult();

exports.encrypt(...writeBytes(plaintext), ...writeBytes(dek));
const encrypted = readResult();
assert("encrypt returns data", encrypted.length > 0);
assert("encrypt not an error", !isError(encrypted));

// Encrypted format: [12-byte nonce][ciphertext]
const encNonce = encrypted.slice(0, 12);
const ciphertext = encrypted.slice(12);
assert("nonce is 12 bytes", encNonce.length === 12);
assert("ciphertext is longer than plaintext (includes auth tag)", ciphertext.length > plaintext.length);

exports.decrypt(...writeBytes(ciphertext), ...writeBytes(encNonce), ...writeBytes(dek));
const decrypted = readResult();
assert("decrypt recovers plaintext", !isError(decrypted));
assert("plaintext matches", arraysEqual(decrypted, plaintext));

// Wrong key fails
exports.generateKey();
const wrongKey = readResult();
exports.decrypt(...writeBytes(ciphertext), ...writeBytes(encNonce), ...writeBytes(wrongKey));
const badDecrypt = readResult();
assert("wrong key returns error", isError(badDecrypt));

console.log("\n=== Key Wrapping ===");

exports.generateKey();
const testDek = readResult();

exports.wrapKey(...writeBytes(testDek), ...writeBytes(kek));
const wrapped = readResult();
assert("wrapKey returns data", wrapped.length > 0);
assert("wrapKey not an error", !isError(wrapped));

const wrapNonce = wrapped.slice(0, 12);
const wrappedCiphertext = wrapped.slice(12);

exports.unwrapKey(...writeBytes(wrappedCiphertext), ...writeBytes(wrapNonce), ...writeBytes(kek));
const unwrapped = readResult();
assert("unwrapKey recovers DEK", arraysEqual(unwrapped, testDek));

console.log("\n=== Name Hashing ===");

exports.hashName(...writeString("DATABASE_URL"), ...writeBytes(key));
const hash1 = readResult();
assert("hashName returns 32 bytes", hash1.length === 32);

exports.hashName(...writeString("DATABASE_URL"), ...writeBytes(key));
const hash2 = readResult();
assert("hashName is deterministic", arraysEqual(hash1, hash2));

exports.hashName(...writeString("STRIPE_KEY"), ...writeBytes(key));
const hash3 = readResult();
assert("different names → different hashes", !arraysEqual(hash1, hash3));

console.log("\n=== Asymmetric (X25519) ===");

exports.generateKeypair();
const keypair = readResult();
assert("generateKeypair returns 64 bytes", keypair.length === 64);

const pubKey = keypair.slice(0, 32);
const privKey = keypair.slice(32, 64);

const secret = new TextEncoder().encode("shared-item-key-material");
exports.wrapWithPublicKey(...writeBytes(secret), ...writeBytes(pubKey));
const asymEncrypted = readResult();
assert("wrapWithPublicKey returns data", asymEncrypted.length > 0);
assert("wrapWithPublicKey not an error", !isError(asymEncrypted));

exports.unwrapWithPrivateKey(...writeBytes(asymEncrypted), ...writeBytes(privKey));
const asymDecrypted = readResult();
assert("unwrapWithPrivateKey recovers payload", arraysEqual(asymDecrypted, secret));

// Wrong key fails
exports.generateKeypair();
const wrongKeypair = readResult();
const wrongPriv = wrongKeypair.slice(32, 64);
exports.unwrapWithPrivateKey(...writeBytes(asymEncrypted), ...writeBytes(wrongPriv));
const badAsym = readResult();
assert("wrong private key returns error", isError(badAsym));

// --- Summary ---

console.log("\n================================");
const total = pass + fail;
if (fail === 0) {
  console.log(`\x1b[32mAll ${total} WASM tests passed\x1b[0m`);
} else {
  console.log(`\x1b[31m${fail} of ${total} WASM tests failed\x1b[0m`);
  process.exit(1);
}
