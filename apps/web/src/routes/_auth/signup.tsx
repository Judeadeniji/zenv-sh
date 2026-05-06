import { createFileRoute, Link, redirect, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { z } from "zod"
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
import { storageKeys, mutationKeys } from "#/lib/keys"
import { signupSchema, type SignupInput } from "#/lib/schemas/auth"
import { AlertCircle, ArrowRight, Quote } from "lucide-react"

const searchSchema = z.object({
	invite: z.string().optional(),
})

export const Route = createFileRoute("/_auth/signup")({
	validateSearch: searchSchema,
	beforeLoad: async ({ context }) => {
		try {
			await context.queryClient.ensureQueryData(meQueryOptions)
			throw redirect({ to: "/" })
		} catch (e) {
			if (e && typeof e === "object" && "to" in e) throw e
		}
	},
	component: SignupPage,
})

function SignupPage() {
	const navigate = useNavigate()
	const { invite } = Route.useSearch()

	const form = useForm<SignupInput>({
		resolver: zodResolver(signupSchema),
		defaultValues: { name: "", email: "", password: "" },
	})

	const signUp = useMutation({
		mutationKey: mutationKeys.auth.signup,
		mutationFn: async (data: SignupInput) => {
			const result = await authClient.signUp.email(data)
			if (result.error) throw new Error(result.error.message ?? "Sign up failed")
			return result.data
		},
		onSuccess: () => {
			if (invite) {
				sessionStorage.setItem(storageKeys.inviteToken, invite)
			}
			navigate({ to: "/" })
		},
	})

	const handleOAuth = (provider: "github" | "google") => {
		if (invite) {
			sessionStorage.setItem(storageKeys.inviteToken, invite)
		}
		authClient.signIn.social({
			provider,
			callbackURL: window.location.origin,
		})
	}

	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1">
				{/* Left — signup card */}
				<div className="flex flex-1 items-center justify-center px-4 py-8">
					<div className="w-full max-w-100">
						<CardBox>
							<Card className="p-0">
								<CardHeader className="px-6 pt-6 text-center">
									<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-xs font-bold text-primary-foreground">
										z
									</div>
									<CardTitle>Create your account</CardTitle>
									<CardDescription className="text-xs">No credit card required.</CardDescription>
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

									<form onSubmit={form.handleSubmit((data) => signUp.mutate(data))} className="grid gap-3">
										{signUp.error && (
											<Alert variant="danger">
												<AlertCircle />
												<AlertDescription>{signUp.error.message}</AlertDescription>
											</Alert>
										)}

										<div className="space-y-1.5">
											<Label htmlFor="name" className="text-xs">Full name</Label>
											<Input
												id="name"
												placeholder="Enter your full name"
												{...form.register("name")}
												feedback={form.formState.errors.name ? "error" : undefined}
												autoFocus
											/>
											{form.formState.errors.name && (
												<p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
											)}
										</div>

										<div className="space-y-1.5">
											<Label htmlFor="email" className="text-xs">Email address</Label>
											<Input
												id="email"
												type="email"
												placeholder="Enter your email address"
												{...form.register("email")}
												feedback={form.formState.errors.email ? "error" : undefined}
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

										<Button type="submit" variant="solid" isLoading={signUp.isPending} className="mt-1 w-full">
											Continue
											<ArrowRight />
										</Button>
									</form>
								</CardContent>

								<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
									Already have an account?{" "}
									<Link to="/login" className="font-medium text-primary hover:underline">
										Sign in
									</Link>
								</div>
							</Card>
						</CardBox>
					</div>
				</div>

				{/* Right — testimonial (hidden on mobile) */}
				<div className="hidden flex-1 items-center justify-center bg-muted/20 px-12 lg:flex">
					<div className="max-w-md">
						<Quote className="mb-6 size-8 text-muted-foreground/40" />
						<blockquote className="text-xl font-medium leading-relaxed tracking-tight text-foreground">
							Zero-knowledge means we can finally give developers access to production secrets without losing sleep. zEnv is the missing piece.
						</blockquote>
						<div className="mt-8 flex items-center gap-3">
							<div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-sm font-semibold text-muted-foreground">
								JD
							</div>
							<div>
								<p className="text-sm font-medium">Jane Doe</p>
								<p className="text-xs text-muted-foreground">Platform Engineering Lead</p>
							</div>
						</div>
					</div>
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
