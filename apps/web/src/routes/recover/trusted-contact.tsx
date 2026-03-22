import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery, useMutation } from "@tanstack/react-query"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { api } from "#/lib/api-client"
import { generateKeypair } from "@zenv/amnesia"
import { AlertCircle, Clock, Users, XCircle } from "lucide-react"

export const Route = createFileRoute("/recover/trusted-contact")({
	component: TrustedContactRecoveryPage,
})

function TrustedContactRecoveryPage() {
	const [error, setError] = useState("")

	// Check if there's an active recovery request
	const requestQuery = useQuery({
		queryKey: ["recovery", "request"],
		queryFn: async () => {
			const { data, error } = await api.GET("/auth/recovery/request")
			if (error) return null
			return data;
		},
	})

	// Check if user has a trusted contact
	const statusQuery = useQuery({
		queryKey: ["recovery", "status"],
		queryFn: async () => {
			const { data, error } = await api.GET("/auth/recovery/status")
			if (error) throw new Error("Failed to fetch recovery status")
			return data;
		},
	})

	const initiate = useMutation({
		mutationFn: async () => {
			const { publicKey } = await generateKeypair()
			const toBase64 = (bytes: Uint8Array) => {
				let binary = ""
				for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
				return btoa(binary)
			}

			// Store ephemeral private key in localStorage for later
			// (useless without the approved recovery payload)
			// TODO: store privateKey

			const { error } = await api.POST("/auth/recovery/request", {
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
		mutationFn: async () => {
			const { error } = await api.DELETE("/auth/recovery/request")
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
		<div className="flex min-h-screen items-center justify-center px-4">
			<div className="w-full max-w-sm">
				<div className="mb-6 text-center">
					<div className="mx-auto mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
						<Users className="size-4" />
					</div>
					<h1 className="text-lg font-semibold">Trusted Contact Recovery</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						Request help from your trusted contact to recover vault access.
					</p>
				</div>

				<CardBox>
					<Card>
						<CardContent className="pt-5">
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
										size="md"
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
					</Card>
				</CardBox>

				<div className="mt-4 text-center">
					<Link
						to="/recover"
						className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
					>
						Back to recovery options
					</Link>
				</div>
			</div>
		</div>
	)
}
