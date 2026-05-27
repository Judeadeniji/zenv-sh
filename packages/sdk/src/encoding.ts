/**
 * @zenv/sdk — binary encoding helpers.
 *
 * Lightweight, cross-runtime implementations of hex/base64 encoding.
 * No Buffer, no Node-only APIs — works in Node, Deno, Bun, and edge runtimes.
 *
 * The hex lookup table is precomputed at module load time to avoid repeated
 * `toString(16)` calls in hot paths (e.g. hashing every secret name on load).
 */

// Precomputed hex lookup table — 256 entries, e.g. HEX_MAP[255] === "ff"
const HEX_MAP: string[] = new Array(256)
for (let i = 0; i < 256; i++) {
	HEX_MAP[i] = i.toString(16).padStart(2, "0")
}

/**
 * Convert a `Uint8Array` to a lowercase hex string.
 *
 * @example
 * bytesToHex(new Uint8Array([0xde, 0xad, 0xbe, 0xef])) // "deadbeef"
 */
export function bytesToHex(bytes: Uint8Array): string {
	let hex = ""
	const len = bytes.length
	for (let i = 0; i < len; i++) {
		hex += HEX_MAP[bytes[i]!]
	}
	return hex
}

/**
 * Encode a `Uint8Array` to a base64 string.
 * Uses the platform's `btoa()` — available in all modern runtimes.
 */
export function bytesToBase64(bytes: Uint8Array): string {
	const len = bytes.length
	const chars = new Array(len)
	for (let i = 0; i < len; i++) {
		chars[i] = String.fromCharCode(bytes[i]!)
	}
	return btoa(chars.join(""))
}

/**
 * Decode a base64 string to a `Uint8Array`.
 * Uses the platform's `atob()` — available in all modern runtimes.
 */
export function base64ToBytes(b64: string): Uint8Array {
	const binary = atob(b64)
	const len = binary.length
	const bytes = new Uint8Array(len)
	for (let i = 0; i < len; i++) {
		bytes[i] = binary.charCodeAt(i)
	}
	return bytes
}
