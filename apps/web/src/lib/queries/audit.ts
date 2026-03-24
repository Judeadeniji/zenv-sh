import { queryOptions } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys } from "#/lib/keys"

export function auditQueryOptions(
	projectId: string,
	opts?: {
		page?: number
		per_page?: number
		sort_by?: string
		sort_dir?: "asc" | "desc"
		action?: string
		user_id?: string
		result?: string
	},
) {
	return queryOptions({
		queryKey: queryKeys.audit.list(projectId, opts),
		queryFn: async () => {
			const { data, error } = await api().GET("/audit-logs", {
				params: { query: { project_id: projectId, ...opts } as any },
			})
			if (error || !data) throw new Error("Failed to fetch audit logs")
			return data
		},
		enabled: !!projectId,
		staleTime: 10_000,
	})
}
