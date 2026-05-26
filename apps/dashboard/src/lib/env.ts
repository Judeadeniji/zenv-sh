import { createEnv } from "@t3-oss/env-core"
import { z } from "zod/v4"

export const env = createEnv({
	clientPrefix: "VITE_",
	client: {
		VITE_AUTH_URL: z.url().default("http://auth.zenv.localhost:1355"),
		VITE_API_URL: z.url().default("http://api.zenv.localhost:1355/v1"),
	},
	runtimeEnv: import.meta.env,
	emptyStringAsUndefined: true,
})
