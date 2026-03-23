import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { encrypt, decrypt, hashName } from "@zenv/amnesia"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64, fromBase64 } from "#/lib/encoding"
import { useProjectDEK } from "#/lib/queries/projects"

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

/**
 * Create a secret encrypted with the project DEK.
 *
 * Payload format matches CLI: JSON `{name, value}` encrypted as a single blob.
 * This ensures cross-platform compatibility (web ↔ CLI ↔ SDK).
 */
export function useCreateSecret() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.secrets.create,
		mutationFn: async ({
			projectId,
			environment,
			name,
			value,
			projectDEK,
		}: {
			projectId: string
			environment: string
			name: string
			value: string
			projectDEK: Uint8Array
		}) => {
			const nameHashBytes = await hashName(name, projectDEK)
			const nameHash = toBase64(nameHashBytes)

			// Match CLI format: encrypt {name, value} JSON as single payload
			const payload = new TextEncoder().encode(JSON.stringify({ name, value }))
			const { ciphertext, nonce } = await encrypt(payload, projectDEK)

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
			environment,
			nameHash,
		}: {
			projectId: string
			environment: string
			nameHash: string
		}) => {
			const { error } = await api().DELETE("/secrets/{nameHash}", {
				params: {
					path: { nameHash },
					query: { project_id: projectId, environment },
				},
			})
			if (error) throw new Error("Failed to delete secret")
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
		},
	})
}

/**
 * Fetch all secrets for a project+environment and decrypt them client-side.
 *
 * Flow: List → BulkFetch (ciphertext) → decrypt each with project DEK → parse JSON
 */
export function useDecryptedSecrets(projectId: string, environment: string) {
	const { data: projectDEK } = useProjectDEK(projectId)
	const { data: listData } = useQuery(secretsQueryOptions(projectId, environment))

	const secrets = (listData as { secrets?: { name_hash: string }[] })?.secrets ?? []

	return useQuery({
		queryKey: [...queryKeys.secrets.list(projectId), environment, "decrypted"],
		queryFn: async () => {
			if (!projectDEK || secrets.length === 0) return []

			const nameHashes = secrets.map((s) => s.name_hash)

			const { data, error } = await api().POST("/secrets/bulk", {
				body: {
					project_id: projectId,
					environment,
					name_hashes: nameHashes,
				} as never,
			})
			if (error || !data) throw new Error("Failed to fetch secrets")

			const items = (data as { secrets?: { name_hash: string; ciphertext: string; nonce: string; version?: number; updated_at?: string }[] })?.secrets ?? []

			return Promise.all(items.map(async (s) => {
				try {
					const plaintext = await decrypt(fromBase64(s.ciphertext), fromBase64(s.nonce), projectDEK)
					const parsed = JSON.parse(new TextDecoder().decode(plaintext)) as { name: string; value: string }
					return { name_hash: s.name_hash, name: parsed.name, value: parsed.value, version: s.version, updated_at: s.updated_at }
				} catch {
					return { name_hash: s.name_hash, name: s.name_hash.slice(0, 12) + "…", value: "[decrypt error]", version: s.version, updated_at: s.updated_at }
				}
			}))
		},
		enabled: !!projectDEK && secrets.length > 0,
		staleTime: 15_000,
	})
}
