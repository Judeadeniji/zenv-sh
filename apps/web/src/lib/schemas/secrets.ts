import { z } from "zod"

/** Plaintext hints stored on the server (never put secret values here). */
export const secretMetadataFieldsSchema = z.object({
	description: z.string().max(4000).optional().or(z.literal("")),
	/** Comma-separated tags */
	tags_input: z.string().optional().or(z.literal("")),
})

export type SecretMetadataFields = z.infer<typeof secretMetadataFieldsSchema>

/** Merge optional description/tags with auto-detected MIME (always sent). */
export function buildSecretMetadataPayload(
	fields: SecretMetadataFields,
	mimeType: string,
): Record<string, unknown> {
	const desc = fields.description?.trim()
	const tags = (fields.tags_input ?? "")
		.split(",")
		.map((t) => t.trim())
		.filter(Boolean)
	const out: Record<string, unknown> = { mime_type: mimeType.slice(0, 256) }
	if (desc) out.description = desc
	if (tags.length > 0) out.tags = tags
	return out
}

export const createSecretSchema = z.discriminatedUnion("inputMode", [
	z
		.object({
			inputMode: z.literal("text"),
			name: z.string().min(1, "Name is required"),
			value: z.string().min(1, "Value is required"),
		})
		.extend(secretMetadataFieldsSchema.shape),
	z
		.object({
			inputMode: z.literal("file"),
			name: z.string().min(1, "Name is required"),
			value: z.string(),
		})
		.extend(secretMetadataFieldsSchema.shape),
])

export type CreateSecretInput = z.infer<typeof createSecretSchema>

export const updateSecretSchema = z
	.object({
		value: z.string().min(1, "Value is required"),
	})
	.extend(secretMetadataFieldsSchema.shape)

export type UpdateSecretInput = z.infer<typeof updateSecretSchema>

export const createTokenSchema = z.object({
	name: z.string().min(1, "Name is required"),
	permission: z.enum(["read", "read_write"]),
})

export type CreateTokenInput = z.infer<typeof createTokenSchema>

export const inviteMemberSchema = z.object({
	email: z.email("Enter a valid email"),
	role: z.string().refine((r) => ["admin", "senior_dev", "dev", "contractor", "ci_bot"].includes(r), "Invalid role"),
})

export type InviteMemberInput = z.infer<typeof inviteMemberSchema>
