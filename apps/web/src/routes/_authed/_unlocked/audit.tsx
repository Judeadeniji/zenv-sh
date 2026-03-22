import { createFileRoute } from "@tanstack/react-router"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { Shield } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/audit")({
	component: AuditPage,
})

function AuditPage() {
	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-lg font-semibold">Audit Log</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						A record of every action in your organization.
					</p>
				</div>
			</div>

			<Empty className="min-h-80">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<Shield />
					</EmptyMedia>
					<EmptyContent>
						<EmptyTitle>No activity yet</EmptyTitle>
						<EmptyDescription>
							Every secret access, change, and token usage is logged here automatically.
							Activity will appear once you start using your project.
						</EmptyDescription>
					</EmptyContent>
				</EmptyHeader>
			</Empty>
		</div>
	)
}
