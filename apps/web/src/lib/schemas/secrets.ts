import { z } from "zod"

export const createSecretSchema = z.object({
	name: z
		.string()
		.min(1, "Name is required")
		.max(256, "Name too long"),
	value: z.string().min(1, "Value is required"),
})

export type CreateSecretInput = z.infer<typeof createSecretSchema>

export const updateSecretSchema = z.object({
	value: z.string().min(1, "Value is required"),
})

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
