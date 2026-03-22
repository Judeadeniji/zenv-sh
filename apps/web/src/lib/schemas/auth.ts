import { z } from "zod"

export const loginSchema = z.object({
	email: z.string().email("Enter a valid email"),
	password: z.string().min(1, "Password is required"),
})

export const signupSchema = z.object({
	name: z.string().min(1, "Name is required"),
	email: z.string().email("Enter a valid email"),
	password: z.string().min(8, "Password must be at least 8 characters"),
})

export const pinSchema = z.object({
	pin: z
		.string()
		.min(6, "PIN must be at least 6 digits")
		.regex(/^\d+$/, "PIN must contain only digits"),
})

export const passphraseSchema = z.object({
	passphrase: z.string().min(12, "Passphrase must be at least 12 characters"),
})

export const confirmPinSchema = pinSchema.extend({
	confirmPin: z.string(),
}).refine((data) => data.pin === data.confirmPin, {
	message: "PINs don't match",
	path: ["confirmPin"],
})

export const confirmPassphraseSchema = passphraseSchema.extend({
	confirmPassphrase: z.string(),
}).refine((data) => data.passphrase === data.confirmPassphrase, {
	message: "Passphrases don't match",
	path: ["confirmPassphrase"],
})

export type LoginInput = z.infer<typeof loginSchema>
export type SignupInput = z.infer<typeof signupSchema>
export type PinInput = z.infer<typeof pinSchema>
export type PassphraseInput = z.infer<typeof passphraseSchema>
