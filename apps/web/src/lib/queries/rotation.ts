import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"

export function useKeyGrantMembers(projectId: string) {
	return useQuery({
		queryKey: [...queryKeys.projects.detail(projectId), "key-grants"],
		queryFn: async () => {
			const { data, error } = await api().GET("/projects/{projectID}/key-grants", {
				params: { path: { projectID: projectId } },
			})
			if (error || !data) throw new Error(error.error || "Failed to fetch key grants")
			return data.members
		},
		enabled: !!projectId,
	})
}

export function useStartRotation() {
	return useMutation({
		mutationKey: mutationKeys.rotation.start,
		mutationFn: async ({ projectId, totalItems }: { projectId: string; totalItems: number }) => {
			const { data, error } = await api().POST("/projects/{projectID}/rotation/start", {
				params: { path: { projectID: projectId } },
				body: { total_items: totalItems },
			})
			if (error || !data) throw new Error(error.error || "Failed to start rotation")
			return data as { rotation_id: string; status: string }
		},
	})
}

export function useStageRotation() {
	return useMutation({
		mutationKey: mutationKeys.rotation.stage,
		mutationFn: async ({
			projectId,
			rotationId,
			items,
		}: {
			projectId: string
			rotationId: string
			items: { vault_item_id: string; new_ciphertext: string; new_nonce: string }[]
		}) => {
			const { data, error } = await api().POST(
				"/projects/{projectID}/rotation/{rotationID}/stage",
				{
					params: { path: { projectID: projectId, rotationID: rotationId } },
					body: { items },
				},
			)
			if (error || !data) throw new Error(error.error || "Failed to stage rotation items")
			return data as { staged: number; total_staged: number; total: number }
		},
	})
}

export function useCommitRotation() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.rotation.commit,
		mutationFn: async ({
			projectId,
			rotationId,
			newWrappedProjectDek,
			newProjectSalt,
			newKeyGrants,
		}: {
			projectId: string
			rotationId: string
			newWrappedProjectDek: string
			newProjectSalt: string
			newKeyGrants: { user_id: string; wrapped_project_vault_key: string }[]
		}) => {
			const { data, error } = await api().POST(
				"/projects/{projectID}/rotation/{rotationID}/commit",
				{
					params: { path: { projectID: projectId, rotationID: rotationId } },
					body: {
						new_wrapped_project_dek: newWrappedProjectDek,
						new_project_salt: newProjectSalt,
						new_key_grants: newKeyGrants,
					},
				},
			)
			if (error || !data) throw new Error(error.error || "Failed to commit rotation")
			return data
		},
		onSuccess: (_, { projectId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) })
			qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
		},
	})
}

export function useCancelRotation() {
	return useMutation({
		mutationKey: mutationKeys.rotation.cancel,
		mutationFn: async ({ projectId, rotationId }: { projectId: string; rotationId: string }) => {
			const { error } = await api().DELETE(
				"/projects/{projectID}/rotation/{rotationID}",
				{
					params: { path: { projectID: projectId, rotationID: rotationId } },
				},
			)
			if (error) throw new Error(error.error || "Failed to cancel rotation")
		},
	})
}
