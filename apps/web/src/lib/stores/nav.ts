import { create } from "zustand"
import { persist } from "zustand/middleware"
import { storageKeys } from "#/lib/keys"

interface NavState {
	activeOrgId: string | null
	activeProjectId: string | null
	setActiveOrg: (orgId: string) => void
	setActiveProject: (projectId: string) => void
	reset: () => void
}

export const useNavStore = create<NavState>()(
	persist(
		(set) => ({
			activeOrgId: null,
			activeProjectId: null,
			setActiveOrg: (orgId) => set({ activeOrgId: orgId, activeProjectId: null }),
			setActiveProject: (projectId) => set({ activeProjectId: projectId }),
			reset: () => set({ activeOrgId: null, activeProjectId: null }),
		}),
		{ name: storageKeys.nav },
	),
)
