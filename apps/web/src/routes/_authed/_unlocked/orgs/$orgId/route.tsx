import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { orgsQueryOptions } from "#/lib/queries/orgs"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId")({
	beforeLoad: async ({ context, params }) => {
		const orgs = await context.queryClient.ensureQueryData(orgsQueryOptions())
		const orgList = (orgs as { organizations?: { id: string; name: string }[] }).organizations ?? []
		const org = orgList.find((o) => o.id === params.orgId)

		if (!org) {
			throw redirect({ to: "/" })
		}

		return { orgId: params.orgId, org }
	},
	component: () => <Outlet />,
})
