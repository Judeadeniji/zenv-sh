/**
 * Offloads Argon2id key derivation to a Web Worker so the UI stays responsive.
 *
 * Drop-in replacement for `deriveKeys` from @zenv/amnesia — same signature,
 * same return type, but runs off the main thread.
 *
 * Falls back to main-thread derivation on the server (SSR) or if workers
 * are unavailable.
 */
import type { KeyType } from "@zenv/amnesia"

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

export async function deriveKeysAsync(
	vaultKey: string,
	salt: Uint8Array,
	keyType: KeyType,
): Promise<{ kek: Uint8Array; authKey: Uint8Array }> {
	// SSR or no Worker support — fall back to main thread
	if (typeof Worker === "undefined") {
		const { deriveKeys } = await import("@zenv/amnesia")
		return deriveKeys(vaultKey, salt, keyType)
	}

	return new Promise((resolve, reject) => {
		const worker = new Worker(
			new URL("./derive-keys.worker.ts", import.meta.url),
			{ type: "module" },
		)

		worker.onmessage = (e: MessageEvent<{ kek: string; authKey: string }>) => {
			resolve({
				kek: fromBase64(e.data.kek),
				authKey: fromBase64(e.data.authKey),
			})
			worker.terminate()
		}

		worker.onerror = (e) => {
			reject(new Error(e.message || "Worker key derivation failed"))
			worker.terminate()
		}

		worker.postMessage({
			vaultKey,
			salt: toBase64(salt),
			keyType,
		})
	})
}
