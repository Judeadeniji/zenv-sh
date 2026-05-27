/**
 * @zenv/vite-plugin — secret polling and diffing.
 *
 * Handles the background polling loop that checks for secret updates
 * while the Vite dev server is running.
 */

import type { ViteDevServer } from "vite"
import { buildDefineMap, fetchSecrets } from "./loader.ts"
import { log } from "./logger.ts"
import type { ResolvedZEnvPluginOptions } from "./types.ts"

/**
 * Compare two secret dictionaries.
 * Returns `true` if any keys were added, removed, or if any values changed.
 */
export function secretsChanged(
	prev: Record<string, string>,
	next: Record<string, string>,
): boolean {
	const prevKeys = Object.keys(prev)
	const nextKeys = Object.keys(next)

	if (prevKeys.length !== nextKeys.length) return true

	for (const key of prevKeys) {
		if (prev[key] !== next[key]) return true
	}

	return false
}

/**
 * Start a background polling loop to check for secret updates.
 *
 * If changes are detected, the Vite dev server is restarted so that
 * the new `define` values take effect across the entire project.
 *
 * @returns a NodeJS timer handle that can be cleared with `clearInterval`.
 */
export function startPolling(
	server: ViteDevServer,
	opts: ResolvedZEnvPluginOptions,
	currentSecrets: Record<string, string>,
	onSecretsUpdated: (fresh: Record<string, string>, defines: Record<string, string>) => void,
): ReturnType<typeof setInterval> {
	log(`Polling for secret changes every ${opts.watch}s.`)

	return setInterval(async () => {
		try {
			const fresh = await fetchSecrets(opts)

			if (secretsChanged(currentSecrets, fresh)) {
				log("Secrets changed — restarting dev server…")
				const defines = buildDefineMap(fresh, opts.prefix)
				onSecretsUpdated(fresh, defines)

				// Give the log a moment to flush before restarting
				await new Promise<void>((r) => setTimeout(r, 120))
				await server.restart()
			}
		} catch (err) {
			// Don't crash the server if a single poll fails (e.g. network blip)
			const { warn } = await import("./logger.ts")
			warn(`Failed to poll for secret updates: ${err instanceof Error ? err.message : String(err)}`)
		}
	}, opts.watch * 1000)
}
