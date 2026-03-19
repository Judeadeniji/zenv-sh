import { createEnv } from "@t3-oss/env-core";
import { z } from "zod/v4";

export const env = createEnv({
  server: {
    DATABASE_URL: z.url(),
    BETTER_AUTH_SECRET: z.string().min(32),
    BETTER_AUTH_URL: z.url(),
    TRUSTED_ORIGINS: z.string().default(""),
    COOKIE_DOMAIN: z.string().optional(),
    PORT: z
      .string()
      .transform(Number)
      .default(3000),
  },
  runtimeEnv: process.env,
  emptyStringAsUndefined: true,
});
