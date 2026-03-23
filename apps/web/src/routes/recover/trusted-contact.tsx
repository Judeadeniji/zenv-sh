import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery, useMutation } from "@tanstack/react-query"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { api } from "#/lib/api-client"
import { generateKeypair } from "@zenv/amnesia"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { toBase64 } from "#/lib/encoding"
import { AlertCircle, Clock, Users, XCircle } from "lucide-react"

export const Route = createFileRoute("/recover/trusted-contact")({
	component: TrustedContactRecoveryPage,
})

function TrustedContactRecoveryPage() {
	const [error, setError] = useState("")

	const requestQuery = useQuery({
		queryKey: queryKeys.recovery.request,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/request")
			if (error) return null
			return data;
		},
	})

	const statusQuery = useQuery({
		queryKey: queryKeys.recovery.status,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/status")
			if (error) throw new Error("Failed to fetch recovery status")
			return data;
		},
	})

	const initiate = useMutation({
		mutationKey: mutationKeys.recovery.initiate,
		mutationFn: async () => {
			const { publicKey } = await generateKeypair()

			const { error } = await api().POST("/auth/recovery/request", {
				body: { recovery_public_key: toBase64(publicKey) },
			})
			if (error) throw new Error("Failed to initiate recovery request")
		},
		onSuccess: () => {
			requestQuery.refetch()
		},
		onError: (err) => {
			setError(err.message)
		},
	})

	const cancel = useMutation({
		mutationKey: mutationKeys.recovery.cancel,
		mutationFn: async () => {
			const { error } = await api().DELETE("/auth/recovery/request")
			if (error) throw new Error("Failed to cancel request")
		},
		onSuccess: () => {
			requestQuery.refetch()
		},
	})

	if (statusQuery.isLoading || requestQuery.isLoading) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Spinner size="lg" />
			</div>
		)
	}

	const status = statusQuery.data
	const request = requestQuery.data

	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<Users className="size-4" />
								</div>
								<CardTitle>Trusted Contact Recovery</CardTitle>
								<CardDescription className="text-xs">
									Request help from your trusted contact to recover vault access.
								</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								{error && (
									<Alert variant="danger" className="mb-4">
										<AlertCircle />
										<AlertDescription>{error}</AlertDescription>
									</Alert>
								)}

								{!status?.has_contact && (
									<Alert variant="warning">
										<AlertCircle />
										<AlertDescription>
											No trusted contact configured. You can set one up in Settings after unlocking your vault.
										</AlertDescription>
									</Alert>
								)}

								{status?.has_contact && !request && (
									<div className="space-y-3">
										<p className="text-sm text-muted-foreground">
											Your trusted contact is <span className="font-medium text-foreground">{status.contact_email}</span>.
											Initiating recovery starts a 72-hour waiting period.
										</p>
										<Button
											variant="solid"
											size="sm"
											onClick={() => initiate.mutate()}
											isLoading={initiate.isPending}
											className="w-full"
										>
											Request recovery
										</Button>
									</div>
								)}

								{request && (
									<div className="space-y-3">
										<div className="flex items-center gap-2 text-sm">
											<Clock className="size-4 text-muted-foreground" />
											<span>Status: <span className="font-medium">{request.status}</span></span>
										</div>

										{request.status === "pending" && (
											<>
												<p className="text-xs text-muted-foreground">
													Eligible at: {new Date(request.eligible_at!).toLocaleString()}
												</p>
												<Button
													variant="outline"
													size="sm"
													onClick={() => cancel.mutate()}
													isLoading={cancel.isPending}
													className="w-full"
												>
													<XCircle className="size-3.5" />
													Cancel request
												</Button>
											</>
										)}

										{request.status === "approved" && request.has_payload && (
											<Alert variant="success">
												<AlertDescription>
													Your trusted contact has approved the request. Recovery payload is ready.
												</AlertDescription>
											</Alert>
										)}
									</div>
								)}
							</CardContent>

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								<Link to="/recover" className="font-medium text-primary hover:underline">
									Back to recovery options
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
