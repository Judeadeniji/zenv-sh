import { createFileRoute } from "@tanstack/react-router"
import { PageHeader } from "#/components/ui/page-header"
import { useAuthStore } from "#/lib/stores/auth"

export const Route = createFileRoute("/_authed/_unlocked/")({
	component: DashboardHome,
})

function DashboardHome() {
	const { me } = useAuthStore()

	return (
		<div>
			<PageHeader
				title="Dashboard"
				description={`Welcome back${me?.email ? `, ${me.email}` : ""}. Your vault is unlocked.`}
			/>
		</div>
	)
}
