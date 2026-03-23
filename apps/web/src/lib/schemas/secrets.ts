import { z } from "zod"

export const createSecretSchema = z.object({
	name: z
		.string()
		.min(1, "Name is required")
		.max(256, "Name too long"),
	value: z.string().min(1, "Value is required"),
})

export type CreateSecretInput = z.infer<typeof createSecretSchema>

export const createTokenSchema = z.object({
	name: z.string().min(1, "Name is required"),
	permission: z.enum(["read", "read_write"]),
})

export type CreateTokenInput = z.infer<typeof createTokenSchema>

export const inviteMemberSchema = z.object({
	email: z.string().email("Enter a valid email"),
	role: z.enum(["member", "admin"]),
})

export type InviteMemberInput = z.infer<typeof inviteMemberSchema>
