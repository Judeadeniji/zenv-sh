import { wrapKey, unwrapKey } from "@zenv/amnesia"
import { entropyToMnemonic, mnemonicToEntropy } from "@scure/bip39"
import { wordlist } from "@scure/bip39/wordlists/english.js"

/** Number of mnemonic words for NEW kits. 12 words = 128 bits entropy. */
export const MNEMONIC_WORD_COUNT = 12

/**
 * Generate 16 bytes (128 bits) of entropy via Web Crypto CSPRNG.
 * 12 BIP39 words encode exactly 128 bits.
 */
export function generateRecoveryEntropy(): Uint8Array {
	const entropy = new Uint8Array(16)
	crypto.getRandomValues(entropy)
	return entropy
}

/**
 * Encode entropy as BIP39 mnemonic words.
 * 16 bytes → 12 words, 32 bytes → 24 words.
 */
export function entropyToWords(entropy: Uint8Array): string {
	return entropyToMnemonic(entropy, wordlist)
}

/**
 * Decode BIP39 mnemonic words back to entropy bytes.
 * 12 words → 16 bytes, 24 words → 32 bytes.
 * Throws if the mnemonic is invalid.
 */
export function wordsToEntropy(mnemonic: string): Uint8Array {
	return mnemonicToEntropy(mnemonic, wordlist)
}

/**
 * Derive a 32-byte AES-256 wrapping key from recovery entropy.
 *
 * - 16 bytes (12 words, new format): SHA-256 expands to 32 bytes.
 * - 32 bytes (24 words, legacy format): used directly as the key.
 *
 * This ensures backwards compatibility with kits generated before
 * the switch from 24 to 12 words.
 */
export async function entropyToWrappingKey(entropy: Uint8Array): Promise<Uint8Array> {
	if (entropy.length === 32) {
		// Legacy 24-word kit — entropy IS the 256-bit key
		return entropy
	}
	// New 12-word kit — expand 128 bits → 256 bits via SHA-256
	const hash = await crypto.subtle.digest("SHA-256", entropy)
	return new Uint8Array(hash)
}

/**
 * Wrap DEK with recovery entropy → nonce||ciphertext blob for server storage.
 */
export async function wrapDekForRecovery(dek: Uint8Array, entropy: Uint8Array): Promise<Uint8Array> {
	const key = await entropyToWrappingKey(entropy)
	const { ciphertext, nonce } = await wrapKey(dek, key)
	const out = new Uint8Array(nonce.length + ciphertext.length)
	out.set(nonce, 0)
	out.set(ciphertext, nonce.length)
	return out
}

/**
 * Unwrap DEK from nonce||ciphertext blob using recovery entropy.
 * Works with both 12-word (new) and 24-word (legacy) kits.
 * Throws if the entropy is wrong.
 */
export async function unwrapDekFromRecovery(blob: Uint8Array, entropy: Uint8Array): Promise<Uint8Array> {
	const key = await entropyToWrappingKey(entropy)
	const nonce = blob.slice(0, 12)
	const ciphertext = blob.slice(12)
	return unwrapKey(ciphertext, nonce, key)
}
