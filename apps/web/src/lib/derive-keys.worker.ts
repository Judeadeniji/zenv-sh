/**
 * Web Worker that runs Argon2id key derivation off the main thread.
 *
 * Receives: { vaultKey, salt (base64), keyType }
 * Returns:  { kek (base64), authKey (base64) }
 *
 * Uses base64 for transferring Uint8Arrays across the worker boundary.
 */
import { deriveKeys, type KeyType } from "@zenv/amnesia"

interface DeriveRequest {
	vaultKey: string
	salt: string // base64
	keyType: KeyType
}

interface DeriveResponse {
	kek: string // base64
	authKey: string // base64
}

function toBase64(bytes: Uint8Array): string {
	let binary = ""
	for (let i = 0; i < bytes.length; i++) {
		binary += String.fromCharCode(bytes[i])
	}
	return btoa(binary)
}

function fromBase64(b64: string): Uint8Array {
	const binary = atob(b64)
	const bytes = new Uint8Array(binary.length)
	for (let i = 0; i < binary.length; i++) {
		bytes[i] = binary.charCodeAt(i)
	}
	return bytes
}

self.onmessage = async (e: MessageEvent<DeriveRequest>) => {
	const { vaultKey, salt, keyType } = e.data
	const { kek, authKey } = await deriveKeys(vaultKey, fromBase64(salt), keyType)
	const response: DeriveResponse = {
		kek: toBase64(kek),
		authKey: toBase64(authKey),
	}
	self.postMessage(response)
}
