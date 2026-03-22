import { createFileRoute, Link } from "@tanstack/react-router"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { KeyRound, Users } from "lucide-react"

export const Route = createFileRoute("/recover")({
	component: RecoverPage,
})

function RecoverPage() {
	return (
		<div className="flex min-h-screen items-center justify-center px-4">
			<div className="w-full max-w-md">
				<div className="mb-6 text-center">
					<h1 className="text-lg font-semibold">Recover your vault</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						Choose a recovery method to regain access to your secrets.
					</p>
				</div>

				<div className="grid gap-3">
					<Link to="/recover/kit" className="no-underline">
						<CardBox>
							<Card className="cursor-pointer transition-colors hover:bg-muted/50">
								<CardContent className="flex items-start gap-3 py-4">
									<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
										<KeyRound className="size-4" />
									</div>
									<div>
										<p className="text-sm font-medium text-foreground">Recovery Kit</p>
										<p className="mt-0.5 text-xs text-muted-foreground">
											Enter the 24-word recovery phrase from your Recovery Kit PDF.
										</p>
									</div>
								</CardContent>
							</Card>
						</CardBox>
					</Link>

					<Link to="/recover/trusted-contact" className="no-underline">
						<CardBox>
							<Card className="cursor-pointer transition-colors hover:bg-muted/50">
								<CardContent className="flex items-start gap-3 py-4">
									<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
										<Users className="size-4" />
									</div>
									<div>
										<p className="text-sm font-medium text-foreground">Trusted Contact</p>
										<p className="mt-0.5 text-xs text-muted-foreground">
											Request recovery via a trusted contact. 72-hour waiting period.
										</p>
									</div>
								</CardContent>
							</Card>
						</CardBox>
					</Link>
				</div>

				<div className="mt-6 text-center">
					<Link
						to="/login"
						className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
					>
						Back to sign in
					</Link>
				</div>
			</div>
		</div>
	)
}
