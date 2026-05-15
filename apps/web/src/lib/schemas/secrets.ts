import { z } from "zod"

/** Plaintext hints stored on the server (never put secret values here). */
export const secretMetadataFieldsSchema = z.object({
	mime_type: z.string().max(256).optional().or(z.literal("")),
	description: z.string().max(4000).optional().or(z.literal("")),
	/** Comma-separated tags */
	tags_input: z.string().optional().or(z.literal("")),
})

export type SecretMetadataFields = z.infer<typeof secretMetadataFieldsSchema>

export function buildSecretMetadataPayload(fields: SecretMetadataFields): Record<string, unknown> | undefined {
	const mime = fields.mime_type?.trim()
	const desc = fields.description?.trim()
	const tags = (fields.tags_input ?? "")
		.split(",")
		.map((t) => t.trim())
		.filter(Boolean)
	const out: Record<string, unknown> = {}
	if (mime) out.mime_type = mime
	if (desc) out.description = desc
	if (tags.length > 0) out.tags = tags
	if (Object.keys(out).length === 0) return undefined
	return out
}

export const createSecretSchema = z.discriminatedUnion("inputMode", [
	z.object({
		inputMode: z.literal("text"),
		name: z.string().min(1, "Name is required"),
		value: z.string().min(1, "Value is required"),
	}).merge(secretMetadataFieldsSchema),
	z.object({
		inputMode: z.literal("file"),
		name: z.string().min(1, "Name is required"),
		value: z.string(),
	}).merge(secretMetadataFieldsSchema),
])

export type CreateSecretInput = z.infer<typeof createSecretSchema>

export const updateSecretSchema = z.object({
	value: z.string().min(1, "Value is required"),
}).merge(secretMetadataFieldsSchema)

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
