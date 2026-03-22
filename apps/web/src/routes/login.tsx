import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Separator } from "#/components/ui/separator"
import { GitHubIcon, GoogleIcon } from "#/components/oauth-icons"
import { authClient } from "#/lib/auth-client"
import { loginSchema, type LoginInput } from "#/lib/schemas/auth"
import { AlertCircle } from "lucide-react"

export const Route = createFileRoute("/login")({
	component: LoginPage,
})

function LoginPage() {
	const navigate = useNavigate()

	const form = useForm<LoginInput>({
		resolver: zodResolver(loginSchema),
		defaultValues: { email: "", password: "" },
	})

	const signIn = useMutation({
		mutationFn: async (data: LoginInput) => {
			const result = await authClient.signIn.email(data)
			if (result.error) throw new Error(result.error.message ?? "Sign in failed")
			return result.data
		},
		onSuccess: () => navigate({ to: "/" }),
	})

	const handleOAuth = (provider: "github" | "google") => {
		authClient.signIn.social({
			provider,
			callbackURL: window.location.origin,
		})
	}

	return (
		<div className="flex min-h-screen items-center justify-center px-4">
			<div className="w-full max-w-sm">
				<div className="mb-6 text-center">
					<div className="mx-auto mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
						z
					</div>
					<h1 className="text-lg font-semibold">Sign in to zEnv</h1>
					<p className="mt-1 text-sm text-muted-foreground">Zero-knowledge secret manager</p>
				</div>

				<CardBox>
					<Card>
						<CardContent className="pt-5">
							<div className="grid gap-2">
								<Button variant="outline" size="md" onClick={() => handleOAuth("github")} className="w-full">
									<GitHubIcon />
									Continue with GitHub
								</Button>
								<Button variant="outline" size="md" onClick={() => handleOAuth("google")} className="w-full">
									<GoogleIcon />
									Continue with Google
								</Button>
							</div>

							<div className="my-4 flex items-center gap-3">
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
									<Label htmlFor="email">Email</Label>
									<Input
										id="email"
										type="email"
										placeholder="you@example.com"
										{...form.register("email")}
										feedback={form.formState.errors.email ? "error" : undefined}
									/>
									{form.formState.errors.email && (
										<p className="text-xs text-destructive">{form.formState.errors.email.message}</p>
									)}
								</div>

								<div className="space-y-1.5">
									<Label htmlFor="password">Password</Label>
									<Input
										id="password"
										type="password"
										placeholder="Enter password"
										{...form.register("password")}
										feedback={form.formState.errors.password ? "error" : undefined}
									/>
									{form.formState.errors.password && (
										<p className="text-xs text-destructive">{form.formState.errors.password.message}</p>
									)}
								</div>

								<Button type="submit" variant="solid" size="md" isLoading={signIn.isPending} className="mt-1 w-full">
									Sign in
								</Button>
							</form>
						</CardContent>
					</Card>
				</CardBox>

				<p className="mt-4 text-center text-sm text-muted-foreground">
					Don't have an account?{" "}
					<Link to="/signup" className="font-medium text-foreground underline-offset-4 hover:underline">
						Sign up
					</Link>
				</p>
			</div>
		</div>
	)
}
