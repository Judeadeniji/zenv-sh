import { createFileRoute } from "@tanstack/react-router"
import { OrgSection } from "#/components/settings/org-section"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/settings")({
	component: OrgSettingsPage,
})

function OrgSettingsPage() {
	const { orgId } = Route.useParams()

	return (
		<div>
			<div className="mb-6">
				<h1 className="text-xl font-semibold tracking-tight">Organization Settings</h1>
				<p className="mt-1 text-sm text-muted-foreground">
					Manage your organization's name, members, and settings.
				</p>
			</div>

			<OrgSection orgId={orgId} />
		</div>
	)
}
