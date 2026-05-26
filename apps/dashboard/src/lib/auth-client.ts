import { createAuthClient } from "better-auth/react"
import { adminClient, organizationClient, twoFactorClient } from "better-auth/client/plugins"
import { env } from "./env"

export const authClient = createAuthClient({
	baseURL: env.VITE_AUTH_URL,
	plugins: [adminClient(), organizationClient(), twoFactorClient()],
	fetchOptions: {
		credentials: "include",
	},
})

export const { useSession, signIn, signUp, signOut } = authClient
