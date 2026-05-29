/**
 * Offloads Argon2id key derivation to a Web Worker so the UI stays responsive.
 *
 * Drop-in replacement for `deriveKeys` from @zenv-sh/amnesia — same signature,
 * same return type, but runs off the main thread.
 *
 * Falls back to main-thread derivation on the server (SSR) or if workers
 * are unavailable.
 */
import type { KeyType } from "@zenv-sh/amnesia"
import { fromBase64, toBase64 } from "./encoding";

export async function deriveKeysAsync(
	vaultKey: string,
	salt: Uint8Array,
	keyType: KeyType,
): Promise<{ kek: Uint8Array; authKey: Uint8Array }> {
	// SSR or no Worker support — fall back to main thread
	if (typeof Worker === "undefined") {
		const { deriveKeys } = await import("@zenv-sh/amnesia")
		return deriveKeys(vaultKey, salt, keyType)
	}

	return new Promise((resolve, reject) => {
		const worker = new Worker(new URL("./derive-keys.worker.ts", import.meta.url), {
			type: "module",
		})

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
