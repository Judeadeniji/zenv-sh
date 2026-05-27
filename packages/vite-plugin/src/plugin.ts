/**
 * @zenv/vite-plugin — core plugin implementation.
 *
 * Security model:
 *   - token and projectKey live entirely in Node (Vite's plugin host).
 *   - They are used once per fetch cycle to decrypt secrets, then discarded.
 *   - Only the decrypted values land in the bundle via Vite's `define` API.
 *   - The plugin refuses to operate without an explicit schema.
 */

import type { Plugin, ViteDevServer } from "vite"
import { assertCredentials, buildDefineMap, fetchSecrets, resolveOptions } from "./loader.ts"
import { log, warn } from "./logger.ts"
import type { ZEnvPluginOptions } from "./types.ts"
import { startPolling } from "./watcher.ts"

const PLUGIN_NAME = "vite-plugin-zenv"

export function zenvPlugin(rawOptions: ZEnvPluginOptions): Plugin {
	const opts = resolveOptions(rawOptions)

	let secrets: Record<string, string> = {}
	let defineMap: Record<string, string> = {}
	let pollTimer: ReturnType<typeof setInterval> | null = null
	let isBuild = false

	return {
		name: PLUGIN_NAME,
		enforce: "pre", // ensures defines resolve before any user plugin

		async config(_, { command }) {
			isBuild = command === "build"
			assertCredentials(opts)

			try {
				log(`Fetching secrets for environment "${opts.environment}"…`)
				secrets = await fetchSecrets(opts)
				defineMap = buildDefineMap(secrets, opts.prefix)
				const count = Object.keys(secrets).length
				log(`✓ Loaded ${count} secret(s) → ${opts.prefix}*`)
			} catch (err) {
				const msg = err instanceof Error ? err.message : String(err)
				if (isBuild) {
					throw new Error(`[zenv] Failed to load secrets. Build aborted.\n\n${msg}`)
				}
				warn(`Failed to load secrets on startup. Polling will retry every ${opts.watch}s.`)
				warn(`Error: ${msg}`)
			}

			return { define: defineMap }
		},

		configureServer(server: ViteDevServer) {
			if (opts.watch === 0) {
				log("Secret polling disabled. Secrets will only reload on server restart.")
				return
			}

			pollTimer = startPolling(server, opts, secrets, (fresh, defines) => {
				secrets = fresh
				defineMap = defines
			})
		},

		closeBundle() {
			if (pollTimer !== null) {
				clearInterval(pollTimer)
				pollTimer = null
			}
		},
	}
}
