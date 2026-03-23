import { create } from "zustand"
import { persist } from "zustand/middleware"
import { storageKeys } from "#/lib/keys"

/**
 * Navigation preferences store.
 *
 * Only holds client-side view preferences — NOT resource context.
 * Org and project context now live in the URL (/orgs/$orgId/projects/$projectId/...).
 */

export const ENVIRONMENTS = ["development", "staging", "production"] as const

interface NavState {
	activeEnvironment: string
	setActiveEnvironment: (env: string) => void
	reset: () => void
}

export const useNavStore = create<NavState>()(
	persist(
		(set) => ({
			activeEnvironment: "development",
			setActiveEnvironment: (activeEnvironment) => set({ activeEnvironment }),
			reset: () => set({ activeEnvironment: "development" }),
		}),
		{ name: storageKeys.nav },
	),
)
