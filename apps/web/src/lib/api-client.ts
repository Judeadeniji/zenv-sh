import createClient from "openapi-fetch"
import type { paths } from "./api.d.ts"
import { env } from "./env"

export const api = createClient<paths>({
	baseUrl: env.VITE_API_URL,
	credentials: "include",
})

export type { paths }
