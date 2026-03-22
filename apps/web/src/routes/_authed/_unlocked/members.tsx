import { createFileRoute } from "@tanstack/react-router"
import { Button } from "#/components/ui/button"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { Users, UserPlus } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/members")({
	component: MembersPage,
})

function MembersPage() {
	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-lg font-semibold">Members</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						People in your organization.
					</p>
				</div>
			</div>

			<Empty className="min-h-80">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<Users />
					</EmptyMedia>
					<EmptyContent>
						<EmptyTitle>Just you for now</EmptyTitle>
						<EmptyDescription>
							Invite team members to collaborate on this organization.
							Everyone sets up their own vault — no one can see anyone else's Vault Key.
						</EmptyDescription>
					</EmptyContent>
				</EmptyHeader>
				<Button variant="solid">
					<UserPlus />
					Invite a member
				</Button>
			</Empty>
		</div>
	)
}
