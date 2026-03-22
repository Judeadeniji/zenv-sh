import { createFileRoute, Link, useNavigate, useParams } from "@tanstack/react-router"
import { useMutation } from "@tanstack/react-query"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { authClient } from "#/lib/auth-client"
import { storageKeys } from "#/lib/keys"
import { AlertCircle, UserPlus } from "lucide-react"

export const Route = createFileRoute("/join/$token")({
	component: JoinPage,
})

function JoinPage() {
	const { token } = useParams({ from: "/join/$token" })
	const navigate = useNavigate()

	const accept = useMutation({
		mutationFn: async () => {
			const { data: session } = await authClient.getSession()

			if (!session) {
				sessionStorage.setItem(storageKeys.inviteToken, token)
				navigate({ to: "/signup", search: { invite: token } })
				return null
			}

			const result = await authClient.organization.acceptInvitation({
				invitationId: token,
			})

			if (result.error) {
				throw new Error(result.error.message ?? "Failed to accept invitation")
			}

			return result.data
		},
		onSuccess: (data) => {
			if (data) {
				navigate({ to: "/" })
			}
		},
	})

	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<UserPlus className="size-4" />
								</div>
								<CardTitle>Join an organization</CardTitle>
								<CardDescription className="text-xs">
									You've been invited to join a team on zEnv.
								</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								{accept.error && (
									<Alert variant="danger" className="mb-4">
										<AlertCircle />
										<AlertDescription>{accept.error.message}</AlertDescription>
									</Alert>
								)}

								{accept.isPending ? (
									<div className="flex flex-col items-center gap-3 py-4">
										<Spinner size="md" />
										<p className="text-sm text-muted-foreground">Accepting invitation...</p>
									</div>
								) : (
									<Button
										variant="solid"
										onClick={() => accept.mutate()}
										className="w-full"
									>
										Accept invitation
									</Button>
								)}
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
