// WASM bridge for Amnesia — exports crypto functions to JavaScript.
//
// Built with TinyGo: tinygo build -o wasm/amnesia.wasm -target wasm -no-debug ./wasm/
//
// JS side uses the shared linear memory to pass byte slices in/out.
// Each exported function takes pointer+length args and returns results
// via a pre-allocated output buffer.
package main

import (
	"unsafe"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

// resultBuf is a pre-allocated buffer for returning variable-length results to JS.
// JS reads from this after each call. Max 4KB covers any key/ciphertext we produce.
var resultBuf [4096]byte
var resultLen int

// --- Memory helpers for JS interop ---

//export getResultPtr
func getResultPtr() uintptr {
	return uintptr(unsafe.Pointer(&resultBuf[0]))
}

//export getResultLen
func getResultLen() int {
	return resultLen
}

//export allocate
func allocate(size int) uintptr {
	buf := make([]byte, size)
	return uintptr(unsafe.Pointer(&buf[0]))
}

// readBytes reads a byte slice from WASM linear memory given a pointer and length.
func readBytes(ptr uintptr, length int) []byte {
	if length == 0 {
		return nil
	}
	buf := make([]byte, length)
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length)
	copy(buf, src)
	return buf
}

// writeResult copies data into resultBuf and sets resultLen.
func writeResult(data []byte) {
	resultLen = len(data)
	copy(resultBuf[:], data)
}

// writeError writes an error string prefixed with "ERR:" into the result buffer.
func writeError(err error) {
	msg := []byte("ERR:" + err.Error())
	writeResult(msg)
}

// --- Exported Amnesia functions ---

//export generateSalt
func generateSalt() {
	writeResult(amnesia.GenerateSalt())
}

//export generateNonce
func generateNonce() {
	writeResult(amnesia.GenerateNonce())
}

//export generateKey
func generateKey() {
	writeResult(amnesia.GenerateKey())
}

// NOTE: deriveKeys (Argon2id) is NOT exported to WASM.
// Argon2id uses goroutines internally which TinyGo's WASM target does not support.
// The TypeScript SDK must use a JS-side Argon2id implementation (e.g. argon2-browser
// or hash-wasm) for key derivation, then pass the derived KEK to the WASM functions
// for encrypt/decrypt/wrap/unwrap operations.
//
// This is a deliberate split: Argon2id in JS, everything else in Amnesia WASM.

//export encrypt
func encrypt(plaintextPtr uintptr, plaintextLen int, keyPtr uintptr, keyLen int) {
	plaintext := readBytes(plaintextPtr, plaintextLen)
	key := readBytes(keyPtr, keyLen)

	ciphertext, nonce, err := amnesia.Encrypt(plaintext, key)
	if err != nil {
		writeError(err)
		return
	}

	// Return nonce (12 bytes) + ciphertext concatenated.
	out := make([]byte, len(nonce)+len(ciphertext))
	copy(out[:12], nonce)
	copy(out[12:], ciphertext)
	writeResult(out)
}

//export decrypt
func decrypt(ciphertextPtr uintptr, ciphertextLen int, noncePtr uintptr, nonceLen int, keyPtr uintptr, keyLen int) {
	ciphertext := readBytes(ciphertextPtr, ciphertextLen)
	nonce := readBytes(noncePtr, nonceLen)
	key := readBytes(keyPtr, keyLen)

	plaintext, err := amnesia.Decrypt(ciphertext, nonce, key)
	if err != nil {
		writeError(err)
		return
	}

	writeResult(plaintext)
}

//export wrapKey
func wrapKey(dekPtr uintptr, dekLen int, kekPtr uintptr, kekLen int) {
	dek := readBytes(dekPtr, dekLen)
	kek := readBytes(kekPtr, kekLen)

	ciphertext, nonce, err := amnesia.WrapKey(dek, kek)
	if err != nil {
		writeError(err)
		return
	}

	out := make([]byte, len(nonce)+len(ciphertext))
	copy(out[:12], nonce)
	copy(out[12:], ciphertext)
	writeResult(out)
}

//export unwrapKey
func unwrapKey(ciphertextPtr uintptr, ciphertextLen int, noncePtr uintptr, nonceLen int, kekPtr uintptr, kekLen int) {
	ciphertext := readBytes(ciphertextPtr, ciphertextLen)
	nonce := readBytes(noncePtr, nonceLen)
	kek := readBytes(kekPtr, kekLen)

	dek, err := amnesia.UnwrapKey(ciphertext, nonce, kek)
	if err != nil {
		writeError(err)
		return
	}

	writeResult(dek)
}

//export hashName
func hashName(namePtr uintptr, nameLen int, hmacKeyPtr uintptr, hmacKeyLen int) {
	name := string(readBytes(namePtr, nameLen))
	hmacKey := readBytes(hmacKeyPtr, hmacKeyLen)
	writeResult(amnesia.HashName(name, hmacKey))
}

// NOTE: hashAuthKey is NOT exported to WASM.
// It uses Argon2id internally. The SDK handles Auth Key hashing in JS.

//export generateKeypair
func generateKeypair() {
	pub, priv, err := amnesia.GenerateKeypair()
	if err != nil {
		writeError(err)
		return
	}

	// Return pub (32 bytes) + priv (32 bytes) = 64 bytes.
	out := make([]byte, 64)
	copy(out[:32], pub)
	copy(out[32:], priv)
	writeResult(out)
}

//export wrapWithPublicKey
func wrapWithPublicKey(payloadPtr uintptr, payloadLen int, pubKeyPtr uintptr, pubKeyLen int) {
	payload := readBytes(payloadPtr, payloadLen)
	pubKey := readBytes(pubKeyPtr, pubKeyLen)

	ciphertext, err := amnesia.WrapWithPublicKey(payload, pubKey)
	if err != nil {
		writeError(err)
		return
	}

	writeResult(ciphertext)
}

//export unwrapWithPrivateKey
func unwrapWithPrivateKey(ciphertextPtr uintptr, ciphertextLen int, privKeyPtr uintptr, privKeyLen int) {
	ciphertext := readBytes(ciphertextPtr, ciphertextLen)
	privKey := readBytes(privKeyPtr, privKeyLen)

	plaintext, err := amnesia.UnwrapWithPrivateKey(ciphertext, privKey)
	if err != nil {
		writeError(err)
		return
	}

	writeResult(plaintext)
}

func main() {}
