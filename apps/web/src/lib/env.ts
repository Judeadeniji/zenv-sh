import { createEnv } from "@t3-oss/env-core"
import { z } from "zod/v4"

export const env = createEnv({
	clientPrefix: "VITE_",
	client: {
		VITE_AUTH_URL: z.url().default("https://auth.zenv.localhost:1335"),
		VITE_API_URL: z.url().default("https://api.zenv.localhost:1335"),
	},
	runtimeEnv: import.meta.env,
	emptyStringAsUndefined: true,
})
