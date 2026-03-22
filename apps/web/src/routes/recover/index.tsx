import { createFileRoute, Link } from "@tanstack/react-router"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { ActionCard } from "#/components/ui/card"
import { KeyRound, Users, ShieldAlert } from "lucide-react"

export const Route = createFileRoute("/recover/")({
	component: RecoverPage,
})

function RecoverPage() {
	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-md">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<ShieldAlert className="size-4" />
								</div>
								<CardTitle>Recover your vault</CardTitle>
								<CardDescription className="text-xs">
									Choose a recovery method to regain access to your secrets.
								</CardDescription>
							</CardHeader>

							<CardContent className="grid gap-3 px-6 pt-4 pb-6">
								<Link to="/recover/kit" className="no-underline">
									<ActionCard className="flex cursor-pointer items-start gap-3 transition-colors hover:bg-muted/50">
										<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
											<KeyRound className="size-4" />
										</div>
										<div>
											<p className="text-sm font-medium text-foreground">Recovery Kit</p>
											<p className="mt-0.5 text-xs text-muted-foreground">
												Enter the 24-word recovery phrase from your Recovery Kit PDF.
											</p>
										</div>
									</ActionCard>
								</Link>

								<Link to="/recover/trusted-contact" className="no-underline">
									<ActionCard className="flex cursor-pointer items-start gap-3 transition-colors hover:bg-muted/50">
										<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
											<Users className="size-4" />
										</div>
										<div>
											<p className="text-sm font-medium text-foreground">Trusted Contact</p>
											<p className="mt-0.5 text-xs text-muted-foreground">
												Request recovery via a trusted contact. 72-hour waiting period.
											</p>
										</div>
									</ActionCard>
								</Link>
							</CardContent>

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								<Link to="/login" className="font-medium text-primary hover:underline">
									Back to sign in
								</Link>
							</div>
						</Card>
					</CardBox>
				</div>
			</div>

			<footer className="flex items-center justify-between px-6 py-4 text-xs text-muted-foreground">
				<span>&copy; {new Date().getFullYear()} zEnv</span>
				<div className="flex items-center gap-1">
					<a href="/support" className="hover:text-foreground">Support</a>
					<span>&middot;</span>
					<a href="/privacy" className="hover:text-foreground">Privacy</a>
					<span>&middot;</span>
					<a href="/terms" className="hover:text-foreground">Terms</a>
				</div>
			</footer>
		</div>
	)
}
