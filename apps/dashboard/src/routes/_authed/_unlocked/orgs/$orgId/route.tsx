import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { orgQueryOptions } from "#/lib/queries/orgs"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId")({
	beforeLoad: async ({ context, params }) => {
		const org = await context.queryClient.ensureQueryData(orgQueryOptions(params.orgId))

		if (!org) {
			throw redirect({ to: "/" })
		}

		return { orgId: params.orgId, org }
	},
	component: () => <Outlet />,
})
