import { queryOptions } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys } from "#/lib/keys"

export function auditQueryOptions(projectId: string, opts?: { page?: number; perPage?: number }) {
	const page = opts?.page ?? 1
	const perPage = opts?.perPage ?? 50

	return queryOptions({
		queryKey: [...queryKeys.audit.list(projectId), page, perPage],
		queryFn: async () => {
			const { data, error } = await api().GET("/audit-logs", {
				params: { query: { project_id: projectId, page, per_page: perPage } as never },
			})
			if (error || !data) throw new Error("Failed to fetch audit logs")
			return data
		},
		enabled: !!projectId,
		staleTime: 10_000,
	})
}
