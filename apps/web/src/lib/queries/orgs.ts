import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"

export const orgsQueryOptions = queryOptions({
	queryKey: queryKeys.orgs.all,
	queryFn: async () => {
		const { data, error } = await api().GET("/orgs")
		if (error || !data) throw new Error("Failed to fetch organizations")
		return data
	},
	staleTime: 30_000,
})

export function orgQueryOptions(orgId: string) {
	return queryOptions({
		queryKey: queryKeys.orgs.detail(orgId),
		queryFn: async () => {
			const { data, error } = await api().GET("/orgs/{orgID}", {
				params: { path: { orgID: orgId } },
			})
			if (error || !data) throw new Error("Failed to fetch organization")
			return data
		},
		staleTime: 30_000,
	})
}

export function orgMembersQueryOptions(orgId: string) {
	return queryOptions({
		queryKey: queryKeys.orgs.members(orgId),
		queryFn: async () => {
			const { data, error } = await api().GET("/orgs/{orgID}/members", {
				params: { path: { orgID: orgId } },
			})
			if (error || !data) throw new Error("Failed to fetch members")
			return data
		},
		staleTime: 30_000,
	})
}

export function useCreateOrg() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.orgs.create,
		mutationFn: async ({ name }: { name: string }) => {
			const { data, error } = await api().POST("/orgs", {
				body: { name },
			})
			if (error || !data) throw new Error("Failed to create organization")
			return data
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: queryKeys.orgs.all })
		},
	})
}

export function useAddMember() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.orgs.addMember,
		mutationFn: async ({ orgId, userId }: { orgId: string; userId: string }) => {
			const { data, error } = await api().POST("/orgs/{orgID}/members", {
				params: { path: { orgID: orgId } },
				body: { user_id: userId },
			})
			if (error || !data) throw new Error("Failed to add member")
			return data
		},
		onSuccess: (_, { orgId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.orgs.members(orgId) })
		},
	})
}

export function useRemoveMember() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.orgs.removeMember,
		mutationFn: async ({ orgId, memberId }: { orgId: string; memberId: string }) => {
			const { error } = await api().DELETE("/orgs/{orgID}/members/{memberID}", {
				params: { path: { orgID: orgId, memberID: memberId } },
			})
			if (error) throw new Error("Failed to remove member")
		},
		onSuccess: (_, { orgId }) => {
			qc.invalidateQueries({ queryKey: queryKeys.orgs.members(orgId) })
		},
	})
}
