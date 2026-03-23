import { create } from "zustand"
import { persist } from "zustand/middleware"
import { storageKeys } from "#/lib/keys"

/**
 * Navigation preferences store.
 *
 * Holds client-side view preferences for immediate UI reactivity.
 * Synced to server via usePreferencesSync() in UnlockedLayout.
 */

export const ENVIRONMENTS = ["development", "staging", "production"] as const

interface NavState {
	activeEnvironment: string
	pinnedProjects: string[]
	theme: string

	setActiveEnvironment: (env: string) => void
	pinProject: (id: string) => void
	unpinProject: (id: string) => void
	setTheme: (theme: string) => void
	hydrate: (prefs: { active_environment?: string; pinned_projects?: string[]; theme?: string }) => void
	reset: () => void
}

export const useNavStore = create<NavState>()(
	persist(
		(set) => ({
			activeEnvironment: "development",
			pinnedProjects: [],
			theme: "auto",

			setActiveEnvironment: (activeEnvironment) => set({ activeEnvironment }),

			pinProject: (id) =>
				set((s) => ({
					pinnedProjects: s.pinnedProjects.includes(id)
						? s.pinnedProjects
						: [id, ...s.pinnedProjects],
				})),

			unpinProject: (id) =>
				set((s) => ({
					pinnedProjects: s.pinnedProjects.filter((p) => p !== id),
				})),

			setTheme: (theme) => set({ theme }),

			hydrate: (prefs) =>
				set((s) => ({
					activeEnvironment: prefs.active_environment ?? s.activeEnvironment,
					pinnedProjects: prefs.pinned_projects ?? s.pinnedProjects,
					theme: prefs.theme ?? s.theme,
				})),

			reset: () => set({ activeEnvironment: "development", pinnedProjects: [], theme: "auto" }),
		}),
		{ name: storageKeys.nav },
	),
)
