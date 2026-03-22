import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useAuthStore } from "#/lib/stores/auth"
import { useUnlockVault } from "#/lib/queries/auth"
import { authClient } from "#/lib/auth-client"
import { AlertCircle, Lock } from "lucide-react"

export const Route = createFileRoute("/_authed/unlock")({
	component: UnlockPage,
})

const unlockSchema = z.object({
	vaultKey: z.string().min(1, "Required"),
})
type UnlockInput = z.infer<typeof unlockSchema>

function UnlockPage() {
	const navigate = useNavigate()
	const me = useAuthStore((s) => s.me)
	const isPin = me?.vault_key_type === "pin"

	const form = useForm<UnlockInput>({
		resolver: zodResolver(unlockSchema),
		defaultValues: { vaultKey: "" },
	})

	const unlock = useUnlockVault()

	const onSubmit = (data: UnlockInput) => {
		unlock.mutate(
			{ vaultKey: data.vaultKey },
			{
				onSuccess: () => navigate({ to: "/" }),
				onError: () => form.reset(),
			},
		)
	}

	const handleSignOut = () => {
		authClient.signOut()
		useAuthStore.getState().reset()
		navigate({ to: "/login" })
	}

	return (
		<div className="flex min-h-screen items-center justify-center px-4">
			<div className="w-full max-w-sm">
				<div className="mb-6 text-center">
					<div className="mx-auto mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
						<Lock className="size-4" />
					</div>
					<h1 className="text-lg font-semibold">Unlock your vault</h1>
					{me?.email && <p className="mt-1 text-sm text-muted-foreground">{me.email}</p>}
				</div>

				<CardBox>
					<Card>
						<CardContent className="pt-5">
							<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-3">
								{unlock.error && (
									<Alert variant="danger">
										<AlertCircle />
										<AlertDescription>Wrong Vault Key. Please try again.</AlertDescription>
									</Alert>
								)}

								<div className="space-y-1.5">
									<Label htmlFor="vault-key">{isPin ? "PIN" : "Passphrase"}</Label>
									<Input
										id="vault-key"
										type="password"
										inputMode={isPin ? "numeric" : undefined}
										pattern={isPin ? "[0-9]*" : undefined}
										placeholder={isPin ? "Enter your PIN" : "Enter your passphrase"}
										autoFocus
										{...form.register("vaultKey")}
									/>
								</div>

								<Button
									type="submit"
									variant="solid"
									size="md"
									isLoading={unlock.isPending}
									loadingText="Deriving keys..."
									className="w-full"
								>
									Unlock
								</Button>
							</form>

							<div className="mt-4 text-center">
								<Link
									to="/recover"
									className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
								>
									Forgot Vault Key?
								</Link>
							</div>
						</CardContent>
					</Card>
				</CardBox>

				<div className="mt-4 text-center">
					<button
						type="button"
						onClick={handleSignOut}
						className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
					>
						Sign out
					</button>
				</div>
			</div>
		</div>
	)
}
