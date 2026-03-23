import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery } from "@tanstack/react-query"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { PasswordInput } from "#/components/ui/password-input"
import { Label } from "#/components/ui/label"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from "#/components/ui/input-otp"
import { useAuthStore } from "#/lib/stores/auth"
import { meQueryOptions, useUnlockVault } from "#/lib/queries/auth"
import { authClient } from "#/lib/auth-client"
import { AlertCircle, Lock } from "lucide-react"
import { type KeyType } from "@zenv/amnesia";

const searchSchema = z.object({
	redirect: z.string().optional(),
})

export const Route = createFileRoute("/_authed/unlock")({
	validateSearch: searchSchema,
	component: UnlockPage,
})

const PIN_LENGTH = 6

const unlockSchema = z.object({
	vaultKey: z.string().min(1, "Required"),
})
type UnlockInput = z.infer<typeof unlockSchema>

function UnlockPage() {
	const navigate = useNavigate()
	const { redirect: redirectTo } = Route.useSearch()

	// Sanitize redirect — never redirect back to unlock itself
	const destination = redirectTo && redirectTo !== "/unlock" && redirectTo !== "/vault-setup" ? redirectTo : "/"

	// Get me from React Query (server-safe) — NOT Zustand
	const { data: me } = useQuery(meQueryOptions)
	const isPin = me?.vault_key_type === "pin"

	const form = useForm<UnlockInput>({
		resolver: zodResolver(unlockSchema),
		defaultValues: { vaultKey: "" },
	})

	const unlock = useUnlockVault()

	const onSubmit = (data: UnlockInput) => {
		if (!me) return
		unlock.mutate(
			{
				vaultKey: data.vaultKey,
				salt: me.salt!,
				keyType: me.vault_key_type as KeyType,
			},
			{
				onSuccess: () => navigate({ to: destination }),
				onError: () => form.setValue("vaultKey", ""),
			},
		)
	}

	const handleSignOut = () => {
		authClient.signOut()
		useAuthStore.getState().lock()
		navigate({ to: "/login" })
	}

	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<Lock className="size-4" />
								</div>
								<CardTitle>Unlock your vault</CardTitle>
								{me?.email && (
									<CardDescription className="text-xs">{me.email}</CardDescription>
								)}
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-3">
									{unlock.error && (
										<Alert variant="danger">
											<AlertCircle />
											<AlertDescription>Wrong Vault Key. Please try again.</AlertDescription>
										</Alert>
									)}

									{isPin ? (
										<Controller
											control={form.control}
											name="vaultKey"
											render={({ field }) => (
												<div>
													<InputOTP
														maxLength={PIN_LENGTH}
														value={field.value}
														onChange={(val) => {
															field.onChange(val)
															if (val.length === PIN_LENGTH) {
																form.handleSubmit(onSubmit)()
															}
														}}
														inputMode="numeric"
														pattern="[0-9]*"
														autoFocus
														containerClassName="justify-center w-full mt-3"
														textAlign="center"
														pushPasswordManagerStrategy="none"
													>
														<InputOTPGroup>
															<InputOTPSlot index={0} masked className="size-10 text-lg" />
															<InputOTPSlot index={1} masked className="size-10 text-lg" />
															<InputOTPSlot index={2} masked className="size-10 text-lg" />
														</InputOTPGroup>
														<InputOTPSeparator />
														<InputOTPGroup>
															<InputOTPSlot index={3} masked className="size-10 text-lg" />
															<InputOTPSlot index={4} masked className="size-10 text-lg" />
															<InputOTPSlot index={5} masked className="size-10 text-lg" />
														</InputOTPGroup>
													</InputOTP>
												</div>
											)}
										/>
									) : (
										<div className="space-y-1.5">
											<Label htmlFor="vault-key" className="text-xs">Passphrase</Label>
											<PasswordInput
												id="vault-key"
												placeholder="Enter your passphrase"
												autoFocus
												{...form.register("vaultKey")}
											/>
										</div>
									)}

									{!isPin && (
										<Button
											type="submit"
											variant="solid"
											isLoading={unlock.isPending}
											loadingText="Deriving keys..."
											className="mt-1 w-full"
										>
											Unlock
										</Button>
									)}

									{isPin && unlock.isPending && (
										<p className="text-center text-xs text-muted-foreground">Deriving keys...</p>
									)}
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

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								Not you?{" "}
								<button
									type="button"
									onClick={handleSignOut}
									className="font-medium text-primary hover:underline"
								>
									Sign out
								</button>
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
