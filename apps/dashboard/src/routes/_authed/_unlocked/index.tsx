import { createFileRoute, redirect } from "@tanstack/react-router"
import { orgsQueryOptions } from "#/lib/queries/orgs"

export const Route = createFileRoute("/_authed/_unlocked/")({
	beforeLoad: async ({ context }) => {
		const orgs = await context.queryClient.ensureQueryData(orgsQueryOptions())
		if (orgs.length > 0) {
			throw redirect({
				to: "/orgs/$orgId",
				params: { orgId: orgs[0].id },
			})
		}
		throw redirect({ to: "/onboarding" })
	},
})
