import { z } from "zod"

export const createOrgSchema = z.object({
	name: z
		.string()
		.min(1, "Organization name is required")
		.max(100, "Name must be 100 characters or less"),
})

export const createProjectSchema = z.object({
	name: z
		.string()
		.min(1, "Project name is required")
		.max(100, "Name must be 100 characters or less")
		.regex(/^[a-z0-9][a-z0-9-]*$/, "Lowercase letters, numbers, and hyphens only"),
})

export const inviteMemberSchema = z.object({
	email: z.string().email("Enter a valid email"),
	role: z.enum(["admin", "senior_dev", "dev", "contractor", "ci_bot"]),
})

export type CreateOrgInput = z.infer<typeof createOrgSchema>
export type CreateProjectInput = z.infer<typeof createProjectSchema>
export type InviteMemberInput = z.infer<typeof inviteMemberSchema>
