import {
	encrypt,
	decrypt,
	wrapKey,
	wrapWithPublicKey,
	generateKey,
	generateSalt,
} from "@zenv/amnesia"
import { deriveKeysAsync } from "#/lib/derive-keys"
import { api } from "#/lib/api-client"
import { toBase64, fromBase64, pack } from "#/lib/encoding"

const ENVIRONMENTS = ["development", "staging", "production"]
const BATCH_SIZE = 50

export type RotationPhase = "preparing" | "staging" | "committing" | "complete" | "error"

export interface RotationProgress {
	phase: RotationPhase
	staged: number
	total: number
	error?: string
}

interface RotationParams {
	projectId: string
	oldProjectDEK: Uint8Array
	members: { user_id: string; public_key: string }[]
	onProgress: (progress: RotationProgress) => void
}

interface BulkSecret {
	id: string
	name_hash: string
	ciphertext: string
	nonce: string
	environment: string
}

/**
 * Orchestrates a full DEK rotation:
 *   1. Generate new crypto material
 *   2. Fetch all secrets across all environments
 *   3. Start rotation via API
 *   4. Re-encrypt and stage in batches
 *   5. Commit with new wrapped DEK + key grants
 */
export async function rotateProjectDEK({
	projectId,
	oldProjectDEK,
	members,
	onProgress,
}: RotationParams): Promise<void> {
	let rotationId: string | null = null

	try {
		// ── 1. Generate new crypto material ──
		onProgress({ phase: "preparing", staged: 0, total: 0 })

		const newProjectDEK = generateKey()
		const newSalt = generateSalt()

		// Generate new Project Vault Key and derive KEK from it
		const newProjectVaultKeyBytes = generateKey()
		const newProjectVaultKey = toBase64(newProjectVaultKeyBytes)
		const { kek: newKEK } = await deriveKeysAsync(newProjectVaultKey, newSalt, "passphrase")

		// ── 2. Fetch all secrets across all environments ──
		const allSecrets: BulkSecret[] = []

		for (const env of ENVIRONMENTS) {
			const { data: listData } = await api().GET("/secrets", {
				params: { query: { project_id: projectId, environment: env } },
			})

			const secrets = (listData as { secrets?: { name_hash: string }[] })?.secrets ?? []
			if (secrets.length === 0) continue

			const { data: bulkData } = await api().POST("/secrets/bulk", {
				body: {
					project_id: projectId,
					environment: env,
					name_hashes: secrets.map((s) => s.name_hash),
				} as never,
			})

			const items = (bulkData as { secrets?: { id: string; name_hash: string; ciphertext: string; nonce: string }[] })?.secrets ?? []
			for (const item of items) {
				allSecrets.push({ ...item, environment: env })
			}
		}

		const total = allSecrets.length
		onProgress({ phase: "preparing", staged: 0, total })

		// ── 3. Start rotation ──
		const { data: startData, error: startErr } = await api().POST(
			"/projects/{projectID}/rotation/start",
			{
				params: { path: { projectID: projectId } },
				body: { total_items: total },
			},
		)
		if (startErr || !startData) throw new Error("Failed to start rotation")

		rotationId = (startData as { rotation_id: string }).rotation_id
		onProgress({ phase: "staging", staged: 0, total })

		// ── 4. Re-encrypt and stage in batches ──
		let staged = 0

		for (let i = 0; i < allSecrets.length; i += BATCH_SIZE) {
			const batch = allSecrets.slice(i, i + BATCH_SIZE)

			const items = await Promise.all(
				batch.map(async (secret) => {
					const plaintext = await decrypt(
						fromBase64(secret.ciphertext),
						fromBase64(secret.nonce),
						oldProjectDEK,
					)
					const { ciphertext, nonce } = await encrypt(plaintext, newProjectDEK)
					return {
						vault_item_id: secret.id,
						new_ciphertext: toBase64(ciphertext),
						new_nonce: toBase64(nonce),
					}
				}),
			)

			const { error: stageErr } = await api().POST(
				"/projects/{projectID}/rotation/{rotationID}/stage" as never,
				{
					params: { path: { projectID: projectId, rotationID: rotationId } },
					body: { items },
				},
			)
			if (stageErr) throw new Error("Failed to stage batch")

			staged += batch.length
			onProgress({ phase: "staging", staged, total })
		}

		// ── 5. Prepare commit payload ──
		onProgress({ phase: "committing", staged: total, total })

		// Wrap new DEK with new KEK
		const { ciphertext: wdCt, nonce: wdNonce } = await wrapKey(newProjectDEK, newKEK)
		const wrappedProjectDEK = pack(wdNonce, wdCt)

		// Wrap new Project Vault Key with each member's public key
		const newKeyGrants = members.map((member) => {
			const wrapped = wrapWithPublicKey(
				new TextEncoder().encode(newProjectVaultKey),
				fromBase64(member.public_key),
			)
			return {
				user_id: member.user_id,
				wrapped_project_vault_key: toBase64(wrapped),
			}
		})

		// ── 6. Commit ──
		const { error: commitErr } = await api().POST(
			"/projects/{projectID}/rotation/{rotationID}/commit" as never,
			{
				params: { path: { projectID: projectId, rotationID: rotationId } },
				body: {
					new_wrapped_project_dek: toBase64(wrappedProjectDEK),
					new_project_salt: toBase64(newSalt),
					new_key_grants: newKeyGrants,
				},
			},
		)
		if (commitErr) throw new Error("Failed to commit rotation")

		onProgress({ phase: "complete", staged: total, total })
	} catch (err) {
		const message = err instanceof Error ? err.message : "Rotation failed"
		onProgress({ phase: "error", staged: 0, total: 0, error: message })

		// Attempt cleanup if we have a rotation ID
		if (rotationId) {
			try {
				await api().DELETE(
					"/projects/{projectID}/rotation/{rotationID}" as never,
					{
						params: { path: { projectID: projectId, rotationID: rotationId } },
					},
				)
			} catch {
				// Best-effort cleanup
			}
		}

		throw err
	}
}
