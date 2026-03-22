import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { useAuthStore } from "#/lib/stores/auth"

export const Route = createFileRoute("/_authed/_unlocked")({
	beforeLoad: () => {
		if (!useAuthStore.getState().crypto) {
			throw redirect({ to: "/unlock" })
		}
	},
	component: UnlockedLayout,
})

function UnlockedLayout() {
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
							onClick={() => useAuthStore.getState().lock()}
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
