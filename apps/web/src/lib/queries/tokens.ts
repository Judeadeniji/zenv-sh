import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toast } from "sonner"

export function tokensQueryOptions(
	projectId: string,
	opts?: {
		page?: number
		per_page?: number
		sort_by?: string
		sort_dir?: "asc" | "desc"
		search?: string
		status?: "active" | "revoked" | "all"
	},
) {
	return queryOptions({
		queryKey: queryKeys.tokens.list(projectId, opts),
		queryFn: async () => {
			const { data, error } = await api().GET("/tokens", {
				params: { query: { project_id: projectId, ...opts } },
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
				query: { project_id: projectId },
				body: { name, permission, project_id: projectId, environment },
			})
			if (error || !data) throw new Error("Failed to create token")
			return data
		},
		onSuccess: async (_, { projectId }) => {
			await qc.cancelQueries({ queryKey: queryKeys.tokens.list(projectId) })
			await qc.invalidateQueries({ queryKey: queryKeys.tokens.list(projectId) })
			toast.success("Token created successfully")
		},
		onError: (error) => toast.error(error.message),
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
				params: { path: { tokenID: tokenId } },
				query: { project_id: projectId },
			})
			if (error) throw new Error("Failed to revoke token")
		},
		onSuccess: async (_, { projectId }) => {
			await qc.invalidateQueries({ queryKey: queryKeys.tokens.list(projectId) })
			toast.success("Token revoked successfully")
		},
		onError: (error) => toast.error(error.message),
	})
}

export function useDestroyToken() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.tokens.destroy,
		mutationFn: async ({
			tokenId,
		}: {
			tokenId: string;
			projectId: string;
		}) => {
			const { error } = await api().DELETE("/tokens/{tokenID}/destroy", {
				params: {
					path: { tokenID: tokenId },
				},
			})
			if (error) throw new Error("Failed to delete token")
		},
		onSuccess: async (_, { projectId }) => {
			await qc.invalidateQueries({ queryKey: queryKeys.tokens.list(projectId) })
			toast.success("Token deleted successfully")
		},
		onError: (error) => toast.error(error.message),
	})
}
