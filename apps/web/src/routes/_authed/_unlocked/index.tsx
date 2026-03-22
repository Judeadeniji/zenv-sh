import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { meQueryOptions } from "#/lib/queries/auth"
import { useNavStore } from "#/lib/stores/nav"
import { manageItems } from "#/lib/nav-items"
import { Card, CardContent } from "#/components/ui/card"
import { ArrowRight } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/")({
	component: DashboardHome,
})

function DashboardHome() {
	const { data: me } = useQuery(meQueryOptions)
	const activeProjectId = useNavStore((s) => s.activeProjectId)
	const firstName = me?.email?.split("@")[0] ?? "there"

	return (
		<div>
			<div className="mb-8">
				<h1 className="text-lg font-semibold">Hey, {firstName}</h1>
				<p className="mt-1 text-sm text-muted-foreground">
					Your vault is unlocked. Here's what you can do.
				</p>
			</div>

			{!activeProjectId && (
				<p className="mb-6 rounded-md bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
					Select a project from the sidebar to manage its secrets and tokens.
				</p>
			)}

			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				{manageItems.map((item) => (
					<Link key={item.href} to={item.href} className="group no-underline">
						<Card className="h-full transition-shadow hover:shadow-md">
							<CardContent className="flex flex-col gap-3 p-4">
								<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
									<item.icon className="size-4 text-foreground" />
								</div>
								<div>
									<h3 className="text-sm font-medium">{item.label}</h3>
									<p className="mt-1 text-xs leading-relaxed text-muted-foreground">{item.description}</p>
								</div>
								<span className="mt-auto flex items-center gap-1 text-xs font-medium text-muted-foreground transition-colors group-hover:text-foreground">
									Open <ArrowRight className="size-3" />
								</span>
							</CardContent>
						</Card>
					</Link>
				))}
			</div>
		</div>
	)
}
