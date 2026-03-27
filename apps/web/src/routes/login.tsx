import { createFileRoute, Link, redirect, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { PasswordInput } from "#/components/ui/password-input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Separator } from "#/components/ui/separator"
import { GitHubIcon, GoogleIcon } from "#/components/oauth-icons"
import { authClient } from "#/lib/auth-client"
import { meQueryOptions } from "#/lib/queries/auth"
import { mutationKeys, queryKeys } from "#/lib/keys"
import { loginSchema, type LoginInput } from "#/lib/schemas/auth"
import { AlertCircle, ArrowRight } from "lucide-react"

export const Route = createFileRoute("/login")({
	beforeLoad: async ({ context }) => {
		try {
			await context.queryClient.ensureQueryData(meQueryOptions)
			throw redirect({ to: "/" })
		} catch (e) {
			if (e && typeof e === "object" && "to" in e) throw e
		}
	},
	component: LoginPage,
})

function LoginPage() {
	const navigate = useNavigate()
	const qc = useQueryClient()

	const form = useForm<LoginInput>({
		resolver: zodResolver(loginSchema),
		defaultValues: { email: "", password: "" },
	})

	const signIn = useMutation({
		mutationKey: mutationKeys.auth.login,
		mutationFn: async (data: LoginInput) => {
			const result = await authClient.signIn.email(data)
			if (result.error) throw new Error(result.error.message ?? "Sign in failed")
			return result.data
		},
		onSuccess: () => {
			// Remove stale me-data so the router's beforeLoad always fetches
			// fresh identity for the newly signed-in account.
			qc.removeQueries({ queryKey: queryKeys.auth.me })
			navigate({ to: "/" })
		},
	})

	const handleOAuth = (provider: "github" | "google") => {
		authClient.signIn.social({
			provider,
			callbackURL: window.location.origin,
		})
	}

	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-xs font-bold text-primary-foreground">
									z
								</div>
								<CardTitle>Sign in to zEnv</CardTitle>
								<CardDescription className="text-xs">Welcome back! Please sign in to continue</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								<div className="grid grid-cols-2 gap-2">
									<Button variant="outline" size="sm" onClick={() => handleOAuth("github")}>
										<GitHubIcon />
										GitHub
									</Button>
									<Button variant="outline" size="sm" onClick={() => handleOAuth("google")}>
										<GoogleIcon />
										Google
									</Button>
								</div>

								<div className="my-5 flex items-center gap-3">
									<Separator className="flex-1" />
									<span className="text-xs text-muted-foreground">or</span>
									<Separator className="flex-1" />
								</div>

								<form onSubmit={form.handleSubmit((data) => signIn.mutate(data))} className="grid gap-3">
									{signIn.error && (
										<Alert variant="danger">
											<AlertCircle />
											<AlertDescription>{signIn.error.message}</AlertDescription>
										</Alert>
									)}

									<div className="space-y-1.5">
										<Label htmlFor="email" className="text-xs">Email address</Label>
										<Input
											id="email"
											type="email"
											placeholder="Enter your email address"
											{...form.register("email")}
											feedback={form.formState.errors.email ? "error" : undefined}
											autoFocus
										/>
										{form.formState.errors.email && (
											<p className="text-xs text-destructive">{form.formState.errors.email.message}</p>
										)}
									</div>

									<div className="space-y-1.5">
										<Label htmlFor="password" className="text-xs">Password</Label>
										<PasswordInput
											id="password"
											placeholder="Enter your password"
											{...form.register("password")}
											feedback={form.formState.errors.password ? "error" : undefined}
										/>
										{form.formState.errors.password && (
											<p className="text-xs text-destructive">{form.formState.errors.password.message}</p>
										)}
									</div>

									<Button type="submit" variant="solid" isLoading={signIn.isPending} className="mt-1 w-full">
										Continue
										<ArrowRight />
									</Button>
								</form>
							</CardContent>

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								Don't have an account?{" "}
								<Link to="/signup" className="font-medium text-primary hover:underline">
									Sign up
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
