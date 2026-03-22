import { createFileRoute } from "@tanstack/react-router"
import { Button } from "#/components/ui/button"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "#/components/ui/empty"
import { FileKey, Plus } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/tokens")({
	component: TokensPage,
})

function TokensPage() {
	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-lg font-semibold">Service Tokens</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						Programmatic access for your CI/CD pipelines and applications.
					</p>
				</div>
			</div>

			<Empty className="min-h-80">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<FileKey />
					</EmptyMedia>
					<EmptyContent>
						<EmptyTitle>No service tokens</EmptyTitle>
						<EmptyDescription>
							Service tokens let your applications read secrets without a human in the loop.
							Each token is scoped to a specific project and can be read-only or read-write.
						</EmptyDescription>
					</EmptyContent>
				</EmptyHeader>
				<Button variant="solid">
					<Plus />
					Create a token
				</Button>
			</Empty>
		</div>
	)
}
