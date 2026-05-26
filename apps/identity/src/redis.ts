import Redis from "ioredis"
import { env } from "./env.js"

/** Shared Redis client for Better Auth secondary storage (sessions, rate limits). */
export const redis = new Redis(env.REDIS_URL, {
	// Better Auth issues many sequential commands; avoid ioredis retry limits on each op.
	maxRetriesPerRequest: null,
})
