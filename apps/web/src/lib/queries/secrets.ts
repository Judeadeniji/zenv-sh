import {
	queryOptions,
	useMutation,
	useQuery,
	useQueryClient,
	type QueryClient,
} from "@tanstack/react-query"
import { encrypt, decrypt, hashName } from "@zenv/amnesia"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64, fromBase64 } from "#/lib/encoding"
import type { SecretPayloadKind } from "#/lib/secret-mime"
import { inferTextMime } from "#/lib/secret-mime"
import { resolveSecretMimeBatchServerFn } from "#/server/mime-lookup-fns"

export type { SecretPayloadKind }

/** Plaintext metadata returned with list/bulk (server-visible). */
export type SecretServerMetadata = {
	mime_type?: string
	description?: string
	tags?: string[]
	labels?: Record<string, string>
}

export type DecryptedSecretRow = {
	name_hash: string
	name: string
	value: string
	kind: SecretPayloadKind
	/** Decoded bytes when `kind === "binary"` (encrypted payload used base64). */
	binary?: Uint8Array
	/** MIME from server `mime-types` batch + text sniff fallback (client-only for JSON). */
	resolvedMime?: string
	suggestedDownloadFilename?: string
	version?: number
	updated_at?: string
	metadata?: SecretServerMetadata
}

/** List row + optional decrypted payload (lazy-loaded). */
export type SecretDashboardRow = {
	name_hash: string
	version?: number
	updated_at?: string
	metadata?: SecretServerMetadata
	decrypted?: DecryptedSecretRow
	/** True while this hash is in the eager-fetch set and the payload query has not resolved. */
	isPayloadLoading?: boolean
}

/** Normalize `metadata` from GET /secrets list (JSON object). */
export function parseSecretMetadataFromList(raw: unknown): SecretServerMetadata | undefined {
	if (raw == null) return undefined
	if (typeof raw === "object" && !Array.isArray(raw) && Object.keys(raw as object).length === 0)
		return undefined
	if (typeof raw === "object" && !Array.isArray(raw)) {
		return raw as SecretServerMetadata
	}
	return undefined
}

export function invalidateProjectEnvironmentSecrets(
	qc: QueryClient,
	projectId: string,
	environment: string,
) {
	return qc.invalidateQueries({ queryKey: [...queryKeys.secrets.list(projectId), environment] })
}

type SecretCiphertextRow = {
	name_hash?: string
	ciphertext?: string
	nonce?: string
	version?: number
	updated_at?: string
	metadata?: SecretServerMetadata
}

/** Core decrypt + JSON parse; no MIME batch (use `finalizeDecryptedRowsMime`). */
export async function decryptSecretCiphertextRow(
	s: SecretCiphertextRow,
	projectDEK: Uint8Array,
): Promise<DecryptedSecretRow> {
	const meta = s.metadata
	const mh = s.name_hash ?? ""
	try {
		const plaintext = await decrypt(fromBase64(s.ciphertext!), fromBase64(s.nonce!), projectDEK)
		const parsed = JSON.parse(new TextDecoder().decode(plaintext)) as {
			name: string
			value: string
			type?: string
		}
		if (parsed.type === "base64") {
			return {
				name_hash: s.name_hash!,
				name: parsed.name,
				value: parsed.value,
				kind: "binary" as const,
				binary: fromBase64(parsed.value),
				version: s.version,
				updated_at: s.updated_at,
				metadata: meta && Object.keys(meta).length > 0 ? meta : undefined,
			} satisfies DecryptedSecretRow
		}
		return {
			name_hash: s.name_hash!,
			name: parsed.name,
			value: parsed.value,
			kind: "text" as const,
			version: s.version,
			updated_at: s.updated_at,
			metadata: meta && Object.keys(meta).length > 0 ? meta : undefined,
		} satisfies DecryptedSecretRow
	} catch {
		return {
			name_hash: mh,
			name: `${mh.slice(0, 12)}…`,
			value: "[decrypt error]",
			kind: "text" as const,
			version: s.version,
			updated_at: s.updated_at,
			metadata: meta && Object.keys(meta).length > 0 ? meta : undefined,
		} satisfies DecryptedSecretRow
	}
}

/** Apply server MIME batch + text sniff (same rules as bulk dashboard path). */
export async function finalizeDecryptedRowsMime(
	rows: DecryptedSecretRow[],
): Promise<DecryptedSecretRow[]> {
	const okRows = rows.filter((r) => r.value !== "[decrypt error]")
	const batchInput = okRows.map((r) => ({
		name: r.name,
		kind: r.kind,
		metadataMime: r.metadata?.mime_type,
	}))

	let batch: { mime: string | null; suggestedDownloadFilename: string | null }[] = []
	try {
		batch = await resolveSecretMimeBatchServerFn({ data: { items: batchInput } })
	} catch {
		batch = batchInput.map(() => ({ mime: null, suggestedDownloadFilename: null }))
	}

	let bi = 0
	return rows.map((r) => {
		if (r.value === "[decrypt error]") return r
		const hints = batch[bi++]!
		const resolvedMime =
			hints.mime ?? (r.kind === "text" ? inferTextMime(r.value) : "application/octet-stream")
		return {
			...r,
			resolvedMime,
			suggestedDownloadFilename: hints.suggestedDownloadFilename ?? undefined,
		}
	})
}

export async function decryptSecretPayloadFromApiRow(
	s: SecretCiphertextRow,
	projectDEK: Uint8Array,
): Promise<DecryptedSecretRow> {
	const row = await decryptSecretCiphertextRow(s, projectDEK)
	const [finalized] = await finalizeDecryptedRowsMime([row])
	return finalized!
}

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

export function secretPayloadQueryOptions(params: {
	projectId: string
	environment: string
	nameHash: string
	projectDEK: Uint8Array | undefined
}) {
	const { projectId, environment, nameHash, projectDEK } = params
	return queryOptions({
		queryKey: queryKeys.secrets.payload(projectId, environment, nameHash),
		queryFn: async ({ signal }) => {
			if (!projectDEK) throw new Error("Project DEK required")
			const { data, error } = await api().GET("/secrets/{nameHash}", {
				params: {
					path: { nameHash: toUrlSafeBase64(nameHash) },
					query: { project_id: projectId, environment },
				},
				signal,
			})
			if (error || !data) throw new Error("Failed to fetch secret")
			return decryptSecretPayloadFromApiRow(data as SecretCiphertextRow, projectDEK)
		},
		enabled: !!projectId && !!environment && !!nameHash && !!projectDEK,
		staleTime: 15_000,
		gcTime: 120_000,
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
			value, // string | Uint8Array
			projectDEK,
			metadata,
		}: {
			projectId: string
			environment: string
			name: string
			value: string | Uint8Array
			projectDEK: Uint8Array
			metadata?: Record<string, unknown>
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
					environment: environment as "development" | "staging" | "production",
					name_hash: nameHash,
					ciphertext: toBase64(ciphertext),
					nonce: toBase64(nonce),
					...(metadata && Object.keys(metadata).length > 0 ? { metadata: metadata as never } : {}),
				},
			})
			if (error || !data) throw new Error("Failed to create secret")
			return data
		},
		onSuccess: async (_, { projectId, environment }) => {
			await invalidateProjectEnvironmentSecrets(qc, projectId, environment)
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
					path: { nameHash: toUrlSafeBase64(nameHash) },
					query: { project_id: projectId, environment },
				},
			})
			if (error) throw new Error(error.error || "Failed to delete secret")
		},
		onSuccess: async (_, { projectId, environment }) => {
			await invalidateProjectEnvironmentSecrets(qc, projectId, environment)
		},
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
			metadata,
		}: {
			projectId: string
			environment: string
			nameHash: string
			name: string
			value: string
			projectDEK: Uint8Array
			metadata?: Record<string, unknown>
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
					...(metadata && Object.keys(metadata).length > 0 ? { metadata: metadata as never } : {}),
				},
			})
			if (error || !data) throw new Error("Failed to update secret")
			return data
		},
		onSuccess: async (_, { projectId, environment }) => {
			await invalidateProjectEnvironmentSecrets(qc, projectId, environment)
		},
	})
}

/** Merge-update plaintext metadata only (no ciphertext rotation). */
export function usePatchSecretMetadata() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.secrets.patchMetadata,
		mutationFn: async ({
			projectId,
			environment,
			nameHash,
			patch,
		}: {
			projectId: string
			environment: string
			nameHash: string
			patch: Record<string, unknown>
		}) => {
			const { data, error } = await api().PATCH("/secrets/{nameHash}/metadata", {
				params: {
					path: { nameHash: toUrlSafeBase64(nameHash) },
					query: { project_id: projectId, environment },
				},
				body: patch as never,
			})
			if (error || !data) throw new Error("Failed to update metadata")
			return data
		},
		onSuccess: async (_, { projectId, environment }) => {
			await invalidateProjectEnvironmentSecrets(qc, projectId, environment)
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
			return data as {
				current_version?: number
				versions?: { version?: number; created_at?: string }[]
			}
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
				invalidateProjectEnvironmentSecrets(qc, projectId, environment),
				qc.invalidateQueries({ queryKey: queryKeys.secrets.versions(projectId, nameHash) }),
			])
		},
	})
}
