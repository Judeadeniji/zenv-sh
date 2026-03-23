/**
 * Base64 encoding/decoding for Uint8Array.
 * Uses web-standard APIs that work in both browser and Node.js 18+.
 */

const NONCE_LENGTH = 12

export function toBase64(bytes: Uint8Array): string {
	return btoa(String.fromCodePoint(...bytes))
}

export function fromBase64(str: string): Uint8Array {
	const binary = atob(str)
	const bytes = new Uint8Array(binary.length)
	for (let i = 0; i < binary.length; i++) {
		bytes[i] = binary.charCodeAt(i)
	}
	return bytes
}

/** Pack nonce + ciphertext into a single Uint8Array */
export function pack(nonce: Uint8Array, ciphertext: Uint8Array): Uint8Array {
	const out = new Uint8Array(nonce.length + ciphertext.length)
	out.set(nonce, 0)
	out.set(ciphertext, nonce.length)
	return out
}

/** Unpack nonce + ciphertext from a single Uint8Array */
export function unpack(data: Uint8Array) {
	return {
		nonce: data.slice(0, NONCE_LENGTH),
		ciphertext: data.slice(NONCE_LENGTH),
	}
}
