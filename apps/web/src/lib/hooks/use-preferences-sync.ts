import { useQuery } from "@tanstack/react-query"
import { preferencesQueryOptions, useUpdatePreferences } from "#/lib/queries/preferences"
import { useNavStore } from "#/lib/stores/nav"
import { useRef, useCallback } from "react"

/**
 * Syncs nav store ↔ server preferences.
 *
 * - On mount: hydrates Zustand from server preferences (server wins on first load).
 * - On store changes: debounced PUT to persist back to server.
 *
 * Must be called inside a component that only renders when vault is unlocked.
 */
export function usePreferencesSync() {
	const { data: serverPrefs } = useQuery(preferencesQueryOptions)
	const update = useUpdatePreferences()
	const hydrated = useRef(false)
	const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)

	// Hydrate once from server on first data load
	if (serverPrefs && !hydrated.current) {
		hydrated.current = true
		useNavStore.getState().hydrate(serverPrefs)
	}

	const persist = useCallback(
		(patch: Record<string, unknown>) => {
			if (debounceRef.current) clearTimeout(debounceRef.current)
			debounceRef.current = setTimeout(() => {
				update.mutate(patch as never)
			}, 1000)
		},
		[update],
	)

	return { persist }
}
