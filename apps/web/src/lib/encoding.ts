/**
 * Base64 encoding/decoding for Uint8Array.
 * Uses web-standard APIs that work in both browser and Node.js 18+.
 */

const NONCE_LENGTH = 12

export function toBase64(bytes: Uint8Array): string {
    let binary = '';
    // 32768 is a safe chunk size that stays well below the call stack argument limits of all major browsers
    const chunkSize = 32768; 
    
    for (let i = 0; i < bytes.length; i += chunkSize) {
        const chunk = bytes.subarray(i, i + chunkSize);
        // Array.from is necessary because apply() requires a standard array, not a typed array
        binary += String.fromCharCode.apply(null, Array.from(chunk));
    }
    
    return btoa(binary);
}

export function fromBase64(b64: string): Uint8Array {
	const binary = atob(b64)
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
