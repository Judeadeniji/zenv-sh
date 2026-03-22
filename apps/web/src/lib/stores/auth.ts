import { create } from "zustand"

/**
 * Client-only vault state store.
 *
 * Holds crypto keys in memory (never persisted) and vault lifecycle state.
 * All async data fetching goes through React Query — this store is for
 * client-only state that doesn't come from the server.
 */

export type VaultState = "loading" | "needs-setup" | "locked" | "unlocked"

interface CryptoMaterial {
	kek: Uint8Array
	dek: Uint8Array
	publicKey: Uint8Array
	privateKey: Uint8Array
}

interface MeData {
	email: string
	vault_setup_complete: boolean
	vault_key_type: string
	salt: string
	vault_unlocked: boolean
}

interface AuthState {
	vaultState: VaultState
	me: MeData | null
	crypto: CryptoMaterial | null

	setMe: (me: MeData) => void
	setVaultState: (state: VaultState) => void
	setCrypto: (crypto: CryptoMaterial) => void
	lock: () => void
	reset: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
	vaultState: "loading",
	me: null,
	crypto: null,

	setMe: (me) => set({ me }),
	setVaultState: (vaultState) => set({ vaultState }),
	setCrypto: (crypto) => set({ crypto, vaultState: "unlocked" }),

	lock: () => set({ vaultState: "locked", crypto: null }),

	reset: () => set({ vaultState: "loading", me: null, crypto: null }),
}))
