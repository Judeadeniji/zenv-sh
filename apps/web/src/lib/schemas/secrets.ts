import { z } from "zod"

export const createSecretSchema = z.object({
	name: z
		.string()
		.min(1, "Name is required")
		.regex(/^[A-Z_][A-Z0-9_]*$/, "Use UPPER_SNAKE_CASE (e.g. DATABASE_URL)"),
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
