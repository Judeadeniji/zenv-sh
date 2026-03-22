import { createFileRoute, Outlet, redirect, useNavigate } from "@tanstack/react-router"
import { useAuthStore } from "#/lib/stores/auth"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { useNavStore } from "#/lib/stores/nav"
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
				throw redirect({ to: "/unlock" })
			}
		}

		// Org check — redirect to onboarding if user has zero orgs
		if (location.pathname !== "/onboarding") {
			try {
				const orgs = await context.queryClient.ensureQueryData(orgsQueryOptions)
				const orgList = (orgs as { organizations?: { id: string; name: string }[] }).organizations ?? []

				if (orgList.length === 0) {
					throw redirect({ to: "/onboarding" })
				}

				// Set active org if not already set
				if (typeof window !== "undefined") {
					const nav = useNavStore.getState()
					if (!nav.activeOrgId && orgList[0]?.id) {
						nav.setActiveOrg(orgList[0].id)
					}
				}
			} catch (e) {
				if (e && typeof e === "object" && "to" in e) throw e
			}
		}
	},
	component: UnlockedLayout,
})

function UnlockedLayout() {
	const navigate = useNavigate()
	const crypto = useAuthStore((s) => s.crypto)

	// Client-side crypto gate — catches the SSR case where beforeLoad
	// couldn't check Zustand on the server.
	if (!crypto) {
		navigate({ to: "/unlock" })
		return null
	}

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
