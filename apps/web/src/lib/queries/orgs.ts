import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { authClient } from "../auth-client"
import slugify from "slug"

export function orgsQueryOptions(opts?: {
	page?: number
	per_page?: number
	sort_by?: string
	sort_dir?: "asc" | "desc"
	search?: string
}) {
	return queryOptions({
		queryKey: queryKeys.orgs.list(opts),
		queryFn: async ({ signal }) => {
			const { data, error } = await authClient.organization.list({
				query: opts,
				fetchOptions: { signal },
			})
			if (error || !data) throw new Error(error.message || "Failed to fetch organizations")
			return data
		},
		staleTime: 30_000,
	})
}

export function orgQueryOptions(orgId: string) {
	return queryOptions({
		queryKey: queryKeys.orgs.detail(orgId),
		queryFn: async ({ signal }) => {
			const { data, error } = await api().GET("/orgs/{orgID}", {
				params: { path: { orgID: orgId } },
				signal,
			})
			if (error || !data) throw new Error("Failed to fetch organization")
			return data
		},
		staleTime: 30_000,
	})
}

export function orgMembersQueryOptions(
	orgId: string,
	opts?: {
		page?: number
		per_page?: number
		sort_by?: string
		sort_dir?: "asc" | "desc"
		search?: string
		role?: string
	},
) {
	return queryOptions({
		queryKey: queryKeys.orgs.members(orgId, opts),
		queryFn: async ({ signal }) => {
			const { data, error } = await api().GET("/orgs/{orgID}/members", {
				params: { path: { orgID: orgId }, query: { ...opts } },
				signal,
			})
			if (error || !data) throw new Error(error?.error || "Failed to fetch members")
			return data
		},
		staleTime: 30_000,
	})
}

export function orgInvitationQueries(orgId: string) {
	return queryOptions({
		queryKey: queryKeys.orgs.invitations(orgId),
		queryFn: async ({ signal }) => {
			const data = await authClient.organization.listInvitations({
				query: { organizationId: orgId },
				fetchOptions: { signal, throw: true }
			});

			return data;
		}
	})
}

export function useCreateOrg() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.orgs.create,
		mutationFn: async ({ name }: { name: string }) => {
			const { data, error } = await authClient.organization.create({
				name,
				slug: slugify(name),
				metadata: {
					'__zenv-org': true,
				}
			})
			if (error || !data) throw new Error(error.message || "Failed to create organization")
			return data
		},
		onSuccess: async () => {
			await qc.invalidateQueries({ queryKey: queryKeys.orgs.all })
		},
	})
}

export function useAddMember() {
	const qc = useQueryClient()
	return useMutation({
		mutationKey: mutationKeys.orgs.addMember,
		mutationFn: async ({ orgId, email, role }: { orgId: string; email: string, role: string }) => {
			const { data, error } = await api().POST("/orgs/{orgID}/members", {
				params: { path: { orgID: orgId } },
				body: { email, role },
			})
			if (error || !data) throw new Error(error.error ||"Failed to add member")
			return data
		},
		onSuccess: async (_, { orgId }) => {
			await qc.invalidateQueries({ queryKey: queryKeys.orgs.members(orgId) })
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
			if (error) throw new Error(error.error ||"Failed to remove member")
		},
		onSuccess: async (_, { orgId }) => {
			// Prefix matching queryKeys.orgs.all covers the list and details
			await qc.invalidateQueries({ queryKey: queryKeys.orgs.all })
			await qc.invalidateQueries({ queryKey: queryKeys.orgs.members(orgId) })
		},
	})
}