import { createFileRoute, Outlet, redirect, useNavigate } from "@tanstack/react-router"
import { useAuthStore } from "#/lib/stores/auth"
import { orgsQueryOptions } from "#/lib/queries/orgs"

export const Route = createFileRoute("/_authed/_unlocked")({
	beforeLoad: async ({ context, location }) => {
		// Crypto check is CLIENT-ONLY — on the server we can't know if
		// the vault is unlocked (keys live in browser memory via Zustand).
		// On SSR, skip this check and let the client component handle it.
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
				const orgList = orgs.organizations ?? []

				if (orgList.length === 0) {
					throw redirect({ to: "/onboarding" })
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

	// Client-side crypto gate — if vault is locked after hydration,
	// redirect to unlock. This catches the SSR case where beforeLoad
	// couldn't check Zustand on the server.
	if (!crypto) {
		navigate({ to: "/unlock" })
		return null
	}

	return (
		<div className="min-h-screen">
			<header className="sticky top-0 z-50 border-b border-border bg-background/80 px-4 backdrop-blur-lg">
				<nav className="mx-auto flex max-w-7xl items-center gap-4 py-3">
					<span className="flex items-center gap-2 text-sm font-semibold">
						<span className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-xs font-bold text-primary-foreground">
							z
						</span>
						zEnv
					</span>
					<div className="ml-auto">
						<button
							type="button"
							onClick={() => {
								useAuthStore.getState().lock()
								navigate({ to: "/unlock" })
							}}
							className="text-xs text-muted-foreground hover:text-foreground"
						>
							Lock
						</button>
					</div>
				</nav>
			</header>
			<main className="mx-auto max-w-7xl px-4 py-8">
				<Outlet />
			</main>
		</div>
	)
}
