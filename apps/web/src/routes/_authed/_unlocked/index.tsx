import { createFileRoute, redirect } from "@tanstack/react-router"
import { orgsQueryOptions } from "#/lib/queries/orgs"

export const Route = createFileRoute("/_authed/_unlocked/")({
	beforeLoad: async ({ context }) => {
		const orgs = await context.queryClient.ensureQueryData(orgsQueryOptions)
		const orgList = (orgs as { organizations?: { id: string }[] }).organizations ?? []

		if (orgList.length === 0) {
			throw redirect({ to: "/onboarding" })
		}

		throw redirect({
			to: "/orgs/$orgId",
			params: { orgId: orgList[0].id },
		})
	},
})
