import { create } from "zustand"

/**
 * Client-only crypto key store.
 *
 * Holds vault crypto keys in browser memory (never persisted, never sent to server).
 * This store is ONLY for client-side state that can't exist on the server.
 *
 * Server-available auth state (session, vault_setup_complete, salt, etc.)
 * lives in React Query via meQueryOptions — NOT here.
 */

interface CryptoMaterial {
	kek: Uint8Array
	dek: Uint8Array
	publicKey: Uint8Array
	privateKey: Uint8Array
}

interface AuthState {
	crypto: CryptoMaterial | null
	setCrypto: (crypto: CryptoMaterial) => void
	lock: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
	crypto: null,
	setCrypto: (crypto) => set({ crypto }),
	lock: () => set({ crypto: null }),
}))
