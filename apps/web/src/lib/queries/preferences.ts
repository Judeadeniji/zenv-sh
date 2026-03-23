import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"

export interface UserPreferences {
	pinned_projects?: string[]
	active_environment?: string
	theme?: string
}

export const preferencesQueryOptions = queryOptions({
	queryKey: queryKeys.preferences,
	queryFn: async () => {
		const { data, error } = await api().GET("/preferences" as never)
		if (error) throw new Error("Failed to fetch preferences")
		return (data ?? {}) as UserPreferences
	},
	staleTime: 60_000,
})

export function useUpdatePreferences() {
	const qc = useQueryClient()

	return useMutation({
		mutationKey: mutationKeys.preferences.update,
		mutationFn: async (patch: Partial<UserPreferences>) => {
			const { data, error } = await api().PUT("/preferences", {
				body: patch,
			})
			if (error) throw new Error(error.error || "Failed to update preferences")
			return data as UserPreferences
		},
		onSuccess: (merged) => {
			qc.setQueryData(queryKeys.preferences, merged)
		},
	})
}
