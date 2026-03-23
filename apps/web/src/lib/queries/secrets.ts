import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { encrypt, hashName } from "@zenv/amnesia"
import { api } from "#/lib/api-client"
import { useAuthStore } from "#/lib/stores/auth"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64 } from "#/lib/encoding"

export function secretsQueryOptions(projectId: string, environment: string) {
	return queryOptions({
		queryKey: [...queryKeys.secrets.list(projectId), environment],
		queryFn: async () => {
			const { data, error } = await api().GET("/secrets", {
				params: { query: { project_id: projectId, environment } },
			})
			if (error || !data) throw new Error("Failed to fetch secrets")
			return data
		},
		enabled: !!projectId && !!environment,
		staleTime: 15_000,
	})
}

export function useCreateSecret() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.secrets.create,
		mutationFn: async ({
			projectId,
			environment,
			name,
			value,
		}: {
			projectId: string
			environment: string
			name: string
			value: string
		}) => {
			const crypto = useAuthStore.getState().crypto
			if (!crypto) throw new Error("Vault is locked")

			const nameHashBytes = await hashName(name, crypto.dek)
			const nameHash = toBase64(nameHashBytes)

			const plaintext = new TextEncoder().encode(value)
			const { ciphertext, nonce } = await encrypt(plaintext, crypto.dek)

			const { data, error } = await api().POST("/secrets", {
				body: {
					project_id: projectId,
					environment,
					name_hash: nameHash,
					ciphertext: toBase64(ciphertext),
					nonce: toBase64(nonce),
				} as never,
			})
			if (error || !data) throw new Error("Failed to create secret")
			return data
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
		},
	})
}

export function useDeleteSecret() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.secrets.delete,
		mutationFn: async ({
			projectId,
			nameHash,
		}: {
			projectId: string
			nameHash: string
		}) => {
			const { error } = await api().DELETE("/secrets/{nameHash}", {
				params: {
					path: { nameHash },
					query: { project_id: projectId },
				},
			})
			if (error) throw new Error("Failed to delete secret")
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
		},
	})
}
