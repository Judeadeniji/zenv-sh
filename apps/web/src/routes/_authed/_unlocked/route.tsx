import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { useAuthStore } from "#/lib/stores/auth"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { preferencesQueryOptions } from "#/lib/queries/preferences"
import { usePreferencesSync } from "#/lib/hooks/use-preferences-sync"
import { SidebarProvider, SidebarInset } from "#/components/ui/sidebar"
import { AppSidebar } from "#/components/app-sidebar"
import { AppHeader } from "#/components/app-header"

export const Route = createFileRoute("/_authed/_unlocked")({
	beforeLoad: async ({ context, location }) => {
		// Crypto check is CLIENT-ONLY — on the server we can't know if
		// the vault is unlocked (keys live in browser memory via Zustand).
		if (typeof window !== "undefined") {
			const { crypto } = useAuthStore.getState()
			if (!crypto) {
				throw redirect({
					to: "/unlock",
					search: { redirect: location.pathname },
					hash: location.hash,
				})
			}
		}

		// Onboarding guard — redirect to onboarding if user has zero orgs
		// (skip if already on onboarding to avoid loop)
		if (location.pathname !== "/onboarding") {
			try {
				const orgs = await context.queryClient.ensureQueryData(orgsQueryOptions())
				const orgList = (orgs as { organizations?: { id: string }[] }).organizations ?? []

				if (orgList.length === 0) {
					throw redirect({ to: "/onboarding" })
				}
			} catch (e) {
				if (e && typeof e === "object" && "to" in e) throw e
			}
		}

		// Prefetch preferences so they're ready for hydration.
		context.queryClient.ensureQueryData(preferencesQueryOptions).catch(() => { })
	},
	component: UnlockedLayout,
})

function UnlockedLayout() {
	usePreferencesSync()

	return (
		<SidebarProvider>
			<AppSidebar />
			<SidebarInset>
				<AppHeader />
				<main className="flex-1 px-6 py-6">
					<Outlet />
				</main>
			</SidebarInset>
		</SidebarProvider>
	)
}
