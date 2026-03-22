import { createFileRoute } from "@tanstack/react-router"
import { Button } from "#/components/ui/button"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { KeyRound, Plus, Upload } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/secrets")({
	component: SecretsPage,
})

function SecretsPage() {
	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-lg font-semibold">Secrets</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						Encrypted environment variables for your project.
					</p>
				</div>
			</div>

			{/* Empty state — will be replaced with table when secrets exist */}
			<Empty className="min-h-80">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<KeyRound />
					</EmptyMedia>
					<EmptyContent>
						<EmptyTitle>No secrets yet</EmptyTitle>
						<EmptyDescription>
							Secrets are encrypted on your device before leaving the browser. Not even the server
							can read them. Add your first secret to get started.
						</EmptyDescription>
					</EmptyContent>
				</EmptyHeader>
				<div className="flex gap-2">
					<Button variant="solid">
						<Plus />
						Add a secret
					</Button>
					<Button variant="outline">
						<Upload />
						Import .env
					</Button>
				</div>
			</Empty>
		</div>
	)
}
