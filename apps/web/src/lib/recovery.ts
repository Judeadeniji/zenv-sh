import { generateKey, wrapKey, unwrapKey } from "@zenv/amnesia"
import { entropyToMnemonic, mnemonicToEntropy } from "@scure/bip39"
import { wordlist } from "@scure/bip39/wordlists/english.js"

/**
 * Generate a 32-byte recovery key using Amnesia's CSPRNG.
 */
export function generateRecoveryKey(): Uint8Array {
	return generateKey()
}

/**
 * Encode a 32-byte key as 24 BIP39 mnemonic words.
 */
export function recoveryKeyToMnemonic(key: Uint8Array): string {
	return entropyToMnemonic(key, wordlist)
}

/**
 * Decode 24 BIP39 mnemonic words back to a 32-byte key.
 * Throws if the mnemonic is invalid.
 */
export function mnemonicToRecoveryKey(mnemonic: string): Uint8Array {
	return mnemonicToEntropy(mnemonic, wordlist)
}

/**
 * Wrap DEK with recovery key → nonce||ciphertext blob for server storage.
 */
export async function wrapDekForRecovery(dek: Uint8Array, recoveryKey: Uint8Array): Promise<Uint8Array> {
	const { ciphertext, nonce } = await wrapKey(dek, recoveryKey)
	const out = new Uint8Array(nonce.length + ciphertext.length)
	out.set(nonce, 0)
	out.set(ciphertext, nonce.length)
	return out
}

/**
 * Unwrap DEK from nonce||ciphertext blob using recovery key.
 * Throws if the recovery key is wrong.
 */
export async function unwrapDekFromRecovery(blob: Uint8Array, recoveryKey: Uint8Array): Promise<Uint8Array> {
	const nonce = blob.slice(0, 12)
	const ciphertext = blob.slice(12)
	return unwrapKey(ciphertext, nonce, recoveryKey)
}
