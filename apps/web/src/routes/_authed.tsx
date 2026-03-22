import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { meQueryOptions } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"

export const Route = createFileRoute("/_authed")({
	beforeLoad: async ({ context }) => {
		let me
		try {
			me = await context.queryClient.ensureQueryData(meQueryOptions)
		} catch {
			throw redirect({ to: "/login" })
		}

		useAuthStore.getState().setMe(me)

		if (!me.vault_setup_complete) {
			useAuthStore.getState().setVaultState("needs-setup")
			throw redirect({ to: "/vault-setup" })
		}

		if (!useAuthStore.getState().crypto) {
			useAuthStore.getState().setVaultState("locked")
			throw redirect({ to: "/unlock" })
		}

		useAuthStore.getState().setVaultState("unlocked")
	},
	component: () => <Outlet />,
})
