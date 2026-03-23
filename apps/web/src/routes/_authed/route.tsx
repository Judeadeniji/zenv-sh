import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { meQueryOptions } from "#/lib/queries/auth"

export const Route = createFileRoute("/_authed")({
	beforeLoad: async ({ context, location }) => {
		// Session + vault state check — works on both server and client
		// because meQueryOptions uses the isomorphic API client
		let me
		try {
			me = await context.queryClient.ensureQueryData(meQueryOptions)
		} catch {
			throw redirect({ to: "/login" })
		}

		// Vault not set up → redirect to setup (skip if already there)
		if (!me.vault_setup_complete && location.pathname !== "/vault-setup") {
			throw redirect({ to: "/vault-setup" })
		}

		// Pass me data to child routes via context
		return { me }
	},
	component: () => <Outlet />,
})
