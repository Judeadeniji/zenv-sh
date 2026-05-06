import { queryOptions, useMutation } from "@tanstack/react-query"
import {
	encrypt,
	decrypt,
	wrapKey,
	unwrapKey,
	hashAuthKey,
	generateKeypair,
	generateKey,
	generateSalt,
	type KeyType,
} from "@zenv/amnesia"
import { deriveKeysAsync } from "#/lib/derive-keys"
import { api } from "#/lib/api-client"
import { useAuthStore } from "#/lib/stores/auth"
import { wrapDekForRecovery } from "#/lib/recovery"
import { toBase64, fromBase64, pack, unpack } from "#/lib/encoding"
import { queryKeys, mutationKeys } from "#/lib/keys"

// ── Queries ──

export const meQueryOptions = queryOptions({
	queryKey: queryKeys.auth.me,
	queryFn: async () => {
		const { data, error } = await api().GET("/auth/me")
		if (error || !data) throw new Error("Failed to fetch auth state")
		return data;
	},
	staleTime: 30_000,
})

// ── Mutations ──

export function useSetupVault() {
	return useMutation({
		mutationKey: mutationKeys.auth.setupVault,
		mutationFn: async ({
			vaultKey,
			keyType,
			recoveryKey,
		}: {
			vaultKey: string
			keyType: KeyType
			recoveryKey?: Uint8Array
		}) => {
			const salt = generateSalt()
			const dek = generateKey()
			const { publicKey, privateKey } = await generateKeypair()
			const { kek, authKey } = await deriveKeysAsync(vaultKey, salt, keyType)
			const authKeyHash = await hashAuthKey(authKey)

			const { ciphertext: wdCt, nonce: wdNonce } = await wrapKey(dek, kek)
			const wrappedDEK = pack(wdNonce, wdCt)

			const { ciphertext: wpCt, nonce: wpNonce } = await encrypt(privateKey, dek)
			const wrappedPrivateKey = pack(wpNonce, wpCt)

			const body: Record<string, unknown> = {
				vault_key_type: keyType,
				salt: toBase64(salt),
				auth_key_hash: toBase64(authKeyHash),
				wrapped_dek: toBase64(wrappedDEK),
				public_key: toBase64(publicKey),
				wrapped_private_key: toBase64(wrappedPrivateKey),
			}

			// Wrap DEK for recovery if entropy is provided (16 bytes → SHA-256 → 32-byte key)
			if (recoveryKey) {
				const recoveryBlob = await wrapDekForRecovery(dek, recoveryKey)
				body.recovery_wrapped_dek = toBase64(recoveryBlob)
			}

			const { error } = await api().POST("/auth/setup-vault", { body })
			if (error) throw new Error("Vault setup failed")

			return { kek, dek, publicKey, privateKey }
		},
		onSuccess: (crypto) => {
			useAuthStore.getState().setCrypto(crypto)
		},
	})
}

export function useUnlockVault() {
	return useMutation({
		mutationKey: mutationKeys.auth.unlockVault,
		mutationFn: async ({
			vaultKey,
			salt: saltB64,
			keyType,
		}: {
			vaultKey: string
			salt: string
			keyType: KeyType
		}) => {
			const salt = fromBase64(saltB64)
			const { kek, authKey } = await deriveKeysAsync(vaultKey, salt, keyType)
			const authKeyHash = await hashAuthKey(authKey)

			const { data, error } = await api().POST("/auth/unlock", {
				body: { auth_key_hash: toBase64(authKeyHash) },
			})
			if (error || !data) throw new Error("Wrong Vault Key")

			const res = data as {
				wrapped_dek: string
				wrapped_private_key: string
				public_key: string
			}

			const wd = unpack(fromBase64(res.wrapped_dek))
			const dek = await unwrapKey(wd.ciphertext, wd.nonce, kek)

			const wp = unpack(fromBase64(res.wrapped_private_key))
			const privateKey = await decrypt(wp.ciphertext, wp.nonce, dek)

			const publicKey = fromBase64(res.public_key)

			return { kek, dek, publicKey, privateKey }
		},
		onSuccess: (crypto) => {
			useAuthStore.getState().setCrypto(crypto)
		},
	})
}
