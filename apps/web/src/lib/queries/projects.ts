import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { generateSalt, generateKey, deriveKeys, wrapKey, wrapWithPublicKey } from "@zenv/amnesia"
import { api } from "#/lib/api-client"
import { useAuthStore } from "#/lib/stores/auth"
import { queryKeys, mutationKeys } from "#/lib/keys"

export function projectsQueryOptions(orgId: string) {
	return queryOptions({
		queryKey: queryKeys.projects.list(orgId),
		queryFn: async () => {
			const { data, error } = await api().GET("/projects", {
				params: { query: { organization_id: orgId } },
			})
			if (error || !data) throw new Error("Failed to fetch projects")
			return data
		},
		staleTime: 30_000,
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

function toBase64(bytes: Uint8Array): string {
	let binary = ""
	for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
	return btoa(binary)
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
			const { kek: projectKEK } = await deriveKeys(projectVaultKey, projectSalt, "passphrase")

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
