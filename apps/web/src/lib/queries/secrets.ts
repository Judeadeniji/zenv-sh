import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { encrypt, decrypt, hashName } from "@zenv/amnesia"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64, fromBase64 } from "#/lib/encoding"
import { useProjectDEK } from "#/lib/queries/projects"

export function secretsQueryOptions(projectId: string, environment: string) {
    return queryOptions({
        queryKey: [...queryKeys.secrets.list(projectId), environment],
        queryFn: async ({ signal }) => {
            const { data, error } = await api().GET("/secrets", {
                params: { query: { project_id: projectId, environment } },
                signal,
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
            value,      // string | Uint8Array
            projectDEK,
        }: {
            projectId: string
            environment: string
            name: string
            value: string | Uint8Array
            projectDEK: Uint8Array
        }) => {
            const MAX_BYTES = 1_048_576 // 1 MB

            let jsonPayload: object
            if (typeof value === "string") {
                const encoded = new TextEncoder().encode(value)
                if (encoded.byteLength > MAX_BYTES) throw new Error("Secret exceeds 1 MB limit")
                jsonPayload = { name, value }
            } else {
                if (value.byteLength > MAX_BYTES) throw new Error("File exceeds 1 MB limit")
                jsonPayload = { name, value: toBase64(value), type: "base64" }
            }

            const nameHashBytes = await hashName(name, projectDEK)
            const nameHash = toBase64(nameHashBytes)

            const payload = new TextEncoder().encode(JSON.stringify(jsonPayload))
            const { ciphertext, nonce } = await encrypt(payload, projectDEK)

            const { data, error } = await api().POST("/secrets", {
                body: {
                    project_id: projectId,
                    environment,
                    name_hash: nameHash,
                    ciphertext: toBase64(ciphertext),
                    nonce: toBase64(nonce),
                },
            })
            if (error || !data) throw new Error("Failed to create secret")
            return data
        },
        onSuccess: async (_, { projectId, environment }) => {
            // Prefix matching handles the 'decrypted' query automatically
            await qc.invalidateQueries({ queryKey: [...queryKeys.secrets.list(projectId), environment] })
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
            // Convert standard base64 → URL-safe base64 for path parameter
            const { error } = await api().DELETE("/secrets/{nameHash}", {
                params: {
                    path: { nameHash: toUrlSafeBase64(nameHash) },
                    query: { project_id: projectId, environment },
                },
            })
            if (error) throw new Error(error.error || "Failed to delete secret")
        },
        onSuccess: async (_, { projectId, environment }) => {
            await qc.invalidateQueries({ queryKey: [...queryKeys.secrets.list(projectId), environment] })
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

    const secrets = listData?.secrets ?? []
    // Creating a fingerprint of the name hashes prevents the race condition 
    // where the query runs with stale 'secrets' list data.
    const fingerprint = secrets.map((s) => s.name_hash).join(",")

    return useQuery({
        queryKey: [...queryKeys.secrets.list(projectId), environment, "decrypted", fingerprint],
        queryFn: async ({ signal }) => {
            if (!projectDEK || secrets.length === 0) return []

            const nameHashes = secrets.map((s) => s.name_hash!)

            const { data, error } = await api().POST("/secrets/bulk", {
                body: {
                    project_id: projectId,
                    environment,
                    name_hashes: nameHashes,
                },
                signal,
            })
            if (error || !data) throw new Error("Failed to fetch secrets")

            const items = (data as { secrets?: { name_hash: string; ciphertext: string; nonce: string; version?: number; updated_at?: string }[] })?.secrets ?? []

            return Promise.all(items.map(async (s) => {
                try {
                    const plaintext = await decrypt(fromBase64(s.ciphertext!), fromBase64(s.nonce!), projectDEK)
                    const parsed = JSON.parse(new TextDecoder().decode(plaintext)) as { name: string; value: string }
                    return { name_hash: s.name_hash, name: parsed.name, value: parsed.value, version: s.version, updated_at: s.updated_at }
                } catch {
                    return { name_hash: s.name_hash, name: s.name_hash!.slice(0, 12) + "…", value: "[decrypt error]", version: s.version, updated_at: s.updated_at }
                }
            }))
        },
        enabled: !!projectDEK && secrets.length > 0,
        staleTime: 15_000,
    })
}

// ── Helpers ──

/** Convert standard base64 → URL-safe base64 for use in URL path parameters. */
function toUrlSafeBase64(b64: string): string {
    return b64.replace(/\+/g, "-").replace(/\//g, "_")
}

// ── Update ──

/**
 * Update a secret's value. Re-encrypts {name, value} with the project DEK.
 * The API auto-increments the version and archives the old ciphertext.
 */
export function useUpdateSecret() {
    const qc = useQueryClient()
    return useMutation({
        mutationKey: mutationKeys.secrets.update,
        mutationFn: async ({
            projectId,
            environment,
            nameHash,
            name,
            value,
            projectDEK,
        }: {
            projectId: string
            environment: string
            nameHash: string
            name: string
            value: string
            projectDEK: Uint8Array
        }) => {
            const payload = new TextEncoder().encode(JSON.stringify({ name, value }))
            const { ciphertext, nonce } = await encrypt(payload, projectDEK)

            const { data, error } = await api().PUT("/secrets/{nameHash}", {
                params: {
                    path: { nameHash: toUrlSafeBase64(nameHash) },
                    query: { project_id: projectId, environment },
                },
                body: {
                    ciphertext: toBase64(ciphertext),
                    nonce: toBase64(nonce),
                },
            })
            if (error || !data) throw new Error("Failed to update secret")
            return data
        },
        onSuccess: async (_, { projectId, environment }) => {
            await qc.invalidateQueries({ queryKey: [...queryKeys.secrets.list(projectId), environment] })
        },
    })
}

// ── Versions ──

/** Fetch version history for a secret. */
export function useSecretVersions(projectId: string, environment: string, nameHash: string) {
    return useQuery({
        queryKey: queryKeys.secrets.versions(projectId, nameHash),
        queryFn: async ({ signal }) => {
            const { data, error } = await api().GET("/secrets/{nameHash}/versions", {
                params: {
                    path: { nameHash: toUrlSafeBase64(nameHash) },
                    query: { project_id: projectId, environment },
                },
                signal,
            })
            if (error || !data) throw new Error("Failed to fetch versions")
            return data as { current_version?: number; versions?: { version?: number; created_at?: string }[] }
        },
        enabled: !!projectId && !!environment && !!nameHash,
        staleTime: 10_000,
    })
}

// ── Rollback ──

/** Rollback a secret to a previous version. */
export function useRollbackSecret() {
    const qc = useQueryClient()
    return useMutation({
        mutationKey: mutationKeys.secrets.rollback,
        mutationFn: async ({
            projectId,
            environment,
            nameHash,
            version,
        }: {
            projectId: string
            environment: string
            nameHash: string
            version: number
        }) => {
            const { data, error } = await api().POST("/secrets/{nameHash}/rollback", {
                params: {
                    path: { nameHash: toUrlSafeBase64(nameHash) },
                    query: { project_id: projectId, environment },
                },
                body: { version },
            })
            if (error || !data) throw new Error("Failed to rollback secret")
            return data
        },
        onSuccess: async (_, { projectId, environment, nameHash }) => {
            await Promise.all([
                qc.invalidateQueries({ queryKey: [...queryKeys.secrets.list(projectId), environment] }),
                // Invalidate versions query to refresh current version due to rollback
                qc.invalidateQueries({ queryKey: queryKeys.secrets.versions(projectId, nameHash) })
            ])
        },
    })
}