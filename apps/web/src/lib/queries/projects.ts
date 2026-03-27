import { queryOptions, useMutation, useQuery, useQueryClient, keepPreviousData } from "@tanstack/react-query"
import { generateSalt, generateKey, wrapKey, unwrapKey, wrapWithPublicKey, unwrapWithPrivateKey } from "@zenv/amnesia"
import { deriveKeysAsync } from "#/lib/derive-keys"
import { api } from "#/lib/api-client"
import { useAuthStore } from "#/lib/stores/auth"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64, fromBase64, unpack } from "#/lib/encoding"

export function projectsQueryOptions(
	orgId: string,
	opts?: {
		page?: number
		per_page?: number
		sort_by?: string
		sort_dir?: "asc" | "desc"
		search?: string
	},
) {
	return queryOptions({
		queryKey: queryKeys.projects.list(orgId, opts),
		queryFn: async () => {
			const { data, error } = await api().GET("/projects", {
				params: { query: { organization_id: orgId, ...opts } as any },
			})
			if (error || !data) throw new Error("Failed to fetch projects")
			return data
		},
		staleTime: 30_000,
		placeholderData: keepPreviousData,
	})
}

export function projectQueryOptions(projectId: string) {
	return queryOptions({
		queryKey: queryKeys.projects.detail(projectId),
		queryFn: async () => {
			const { data, error } = await api().GET("/projects/{projectID}", {
				params: { path: { projectID: projectId } },
			})
			if (error || !data) throw new Error("Failed to fetch project")
			return data
		},
		staleTime: 30_000,
	})
}


/**
 * Create a project with client-side crypto material.
 *
 * Generates: project salt, project DEK, wraps DEK with a project KEK derived
 * from a random Project Vault Key, then wraps the Project Vault Key with the
 * user's public key for key grant storage.
 *
 * Returns the plaintext Project Vault Key (shown once at creation).
 */
export function useCreateProject() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.projects.create,
		mutationFn: async ({ name, orgId }: { name: string; orgId: string }) => {
			const { crypto } = useAuthStore.getState()
			if (!crypto) throw new Error("Vault must be unlocked")

			// Generate project crypto material
			const projectSalt = generateSalt()
			const projectDEK = generateKey()

			// Generate a random Project Vault Key and derive a KEK from it
			const projectVaultKeyBytes = generateKey()
			const projectVaultKey = toBase64(projectVaultKeyBytes)
			const { kek: projectKEK } = await deriveKeysAsync(projectVaultKey, projectSalt, "passphrase")

			// Wrap project DEK with project KEK
			const { ciphertext: wdCt, nonce: wdNonce } = await wrapKey(projectDEK, projectKEK)
			const wrappedProjectDEK = new Uint8Array(wdNonce.length + wdCt.length)
			wrappedProjectDEK.set(wdNonce, 0)
			wrappedProjectDEK.set(wdCt, wdNonce.length)

			// Wrap Project Vault Key with user's public key (for key grant)
			const wrappedProjectVaultKey = wrapWithPublicKey(
				new TextEncoder().encode(projectVaultKey),
				crypto.publicKey,
			)

			const { data, error } = await api().POST("/projects", {
				body: {
					name,
					organization_id: orgId,
					project_salt: toBase64(projectSalt),
					wrapped_project_dek: toBase64(wrappedProjectDEK),
					wrapped_project_vault_key: toBase64(wrappedProjectVaultKey),
				},
			})
			if (error || !data) throw new Error("Failed to create project")

			return { ...data, projectVaultKey }
		},
		onSuccess: (_, { orgId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.projects.list(orgId) })
		},
	})
}

export function useDeleteProject() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.projects.delete,
		mutationFn: async ({ projectId }: { projectId: string }) => {
			const { error } = await api().DELETE("/projects/{projectID}" as never, {
				params: { path: { projectID: projectId } },
			} as any)
			if (error) throw new Error("Failed to delete project")
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["projects"] })
		},
	})
}

/**
 * Fetch and unwrap the Project Vault Key from the user's key grant.
 *
 * Flow: GET /projects/{id}/key-grant → wrapped_project_vault_key
 *       → unwrapWithPrivateKey(wrapped, privateKey) → plaintext Project Vault Key
 */
export function useProjectKey(projectId: string) {
	const crypto = useAuthStore((s) => s.crypto)

	return useQuery({
		queryKey: [...queryKeys.projects.detail(projectId), "key-grant"],
		queryFn: async () => {
			if (!crypto) throw new Error("Vault must be unlocked")

			const { data, error } = await api().GET("/projects/{projectID}/key-grant", {
				params: { path: { projectID: projectId } },
			})
			if (error || !data) throw new Error("No key grant found")

			const wrappedBytes = fromBase64(data.wrapped_project_vault_key!)
			const plaintext = unwrapWithPrivateKey(wrappedBytes, crypto.privateKey)
			return new TextDecoder().decode(plaintext)
		},
		enabled: !!crypto && !!projectId,
		staleTime: Number.POSITIVE_INFINITY,
	})
}

/**
 * Derive the Project DEK for encrypting/decrypting secrets.
 *
 * Flow:
 *   1. Unwrap Project Vault Key from key grant (via user's private key)
 *   2. Fetch project_salt + wrapped_project_dek from /projects/{id}/crypto
 *   3. Argon2id(projectVaultKey + projectSalt) → Project KEK
 *   4. AES-256-GCM unwrap(wrapped_project_dek, Project KEK) → Project DEK
 */
export function useProjectDEK(projectId: string) {
	const crypto = useAuthStore((s) => s.crypto)

	return useQuery({
		queryKey: [...queryKeys.projects.detail(projectId), "dek"],
		queryFn: async () => {
			if (!crypto) throw new Error("Vault must be unlocked")

			// 1. Get project vault key from key grant
			const { data: grantData, error: grantErr } = await api().GET("/projects/{projectID}/key-grant", {
				params: { path: { projectID: projectId } },
			})
			if (grantErr || !grantData) throw new Error("No key grant found")

			const wrappedBytes = fromBase64(grantData.wrapped_project_vault_key!)
			const projectVaultKey = new TextDecoder().decode(
				unwrapWithPrivateKey(wrappedBytes, crypto.privateKey),
			)

			// 2. Get project crypto material (salt + wrapped DEK)
			const { data: cryptoData, error: cryptoErr } = await api().GET("/projects/{projectID}/crypto", {
				params: { path: { projectID: projectId } },
			})
			if (cryptoErr || !cryptoData) throw new Error("Project crypto not found")

			const cm = cryptoData
			const projectSalt = fromBase64(cm.project_salt!)
			const wrappedDEK = fromBase64(cm.wrapped_project_dek!)

			// 3. Derive project KEK from project vault key + salt
			const { kek: projectKEK } = await deriveKeysAsync(projectVaultKey, projectSalt, "passphrase")

			// 4. Unwrap project DEK
			const { nonce, ciphertext } = unpack(wrappedDEK)
			const projectDEK = await unwrapKey(ciphertext, nonce, projectKEK)

			return projectDEK
		},
		enabled: !!crypto && !!projectId,
		staleTime: Number.POSITIVE_INFINITY,
	})
}

export interface KeyGrantMember {
	user_id: string
	email: string
	public_key: string // base64
	has_grant: boolean
}

/**
 * All org members for a project with their grant status and public keys.
 * Members without a vault (no public_key) are excluded by the server.
 */
export function listKeyGrantsQueryOptions(projectId: string) {
	return queryOptions({
		queryKey: [...queryKeys.projects.detail(projectId), "key-grants"],
		queryFn: async () => {
			const { data, error } = await api().GET("/projects/{projectID}/key-grants", {
				params: { path: { projectID: projectId } },
			})
			if (error || !data) throw new Error("Failed to fetch key grants")
			return (data.members ?? []) as KeyGrantMember[]
		},
		staleTime: 30_000,
	})
}

/**
 * Batch-upsert key grants for org members who don't have project access yet.
 * Each grant is the Project Vault Key wrapped with that member's public key.
 */
export function useGrantAccess(projectId: string) {
	const qc = useQueryClient()
	return useMutation({
		mutationFn: async (grants: Array<{ user_id: string; wrapped_project_vault_key: string }>) => {
			const { error } = await api().POST("/projects/{projectID}/grants", {
				params: { path: { projectID: projectId } },
				body: { grants },
			})
			if (error) throw new Error((error as { message?: string }).message ?? "Failed to grant access")
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: [...queryKeys.projects.detail(projectId), "key-grants"] })
		},
	})
}

export function projectStatsQueryOptions(projectId: string) {
	return queryOptions({
		queryKey: queryKeys.projects.stats(projectId),
		queryFn: async () => {
			const { data, error } = await api().GET("/projects/{projectID}/stats", {
				params: { path: { projectID: projectId } },
			})
			if (error || !data) throw new Error("Failed to fetch project stats")
			return data
		},
		enabled: !!projectId,
		staleTime: 30_000,
	})
}
