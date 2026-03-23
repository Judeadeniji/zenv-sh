import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"

export function tokensQueryOptions(projectId: string) {
	return queryOptions({
		queryKey: queryKeys.tokens.list(projectId),
		queryFn: async () => {
			const { data, error } = await api().GET("/tokens", {
				params: { query: { project_id: projectId } },
			})
			if (error || !data) throw new Error("Failed to fetch tokens")
			return data
		},
		enabled: !!projectId,
		staleTime: 15_000,
	})
}

export function useCreateToken() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.tokens.create,
		mutationFn: async ({
			projectId,
			name,
			permission,
			environment,
		}: {
			projectId: string
			name: string
			permission: "read" | "read_write"
			environment: string
		}) => {
			const { data, error } = await api().POST("/tokens", {
				params: { query: { project_id: projectId } },
				body: { name, permission, project_id: projectId, environment } as never,
			})
			if (error || !data) throw new Error("Failed to create token")
			return data
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.tokens.list(projectId) })
		},
	})
}

export function useRevokeToken() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.tokens.revoke,
		mutationFn: async ({
			projectId,
			tokenId,
		}: {
			projectId: string
			tokenId: string
		}) => {
			const { error } = await api().DELETE("/tokens/{tokenID}", {
				params: {
					path: { tokenID: tokenId },
					query: { project_id: projectId },
				},
			})
			if (error) throw new Error("Failed to revoke token")
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.tokens.list(projectId) })
		},
	})
}
