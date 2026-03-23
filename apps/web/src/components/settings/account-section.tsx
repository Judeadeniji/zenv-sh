import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQueryClient, useMutation, useQuery } from "@tanstack/react-query"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { PasswordInput } from "#/components/ui/password-input"
import { Label } from "#/components/ui/label"
import { Badge } from "#/components/ui/badge"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { authClient } from "#/lib/auth-client"
import { meQueryOptions } from "#/lib/queries/auth"
import { CheckCircle, AlertCircle, Github, Mail } from "lucide-react"

// ── Schemas ──

const profileSchema = z.object({
	name: z.string().min(1, "Name is required"),
})

type ProfileInput = z.infer<typeof profileSchema>

const passwordSchema = z
	.object({
		currentPassword: z.string().min(1, "Current password is required"),
		newPassword: z.string().min(8, "Password must be at least 8 characters"),
		confirmPassword: z.string(),
	})
	.refine((d) => d.newPassword === d.confirmPassword, {
		message: "Passwords don't match",
		path: ["confirmPassword"],
	})

type PasswordFormInput = z.infer<typeof passwordSchema>

// ── Component ──

export function AccountSection() {
	const { data: me } = useQuery(meQueryOptions)
	const meAny = me as { name?: string; email?: string } | undefined

	return (
		<div>
			<ProfileRow name={meAny?.name ?? ""} email={meAny?.email ?? ""} />
			<SettingsDivider />
			<PasswordRow />
			<SettingsDivider />
			<LinkedAccountsRow />
		</div>
	)
}

// ── Profile ──

function ProfileRow({ name, email }: { name: string; email: string }) {
	const qc = useQueryClient()
	const form = useForm<ProfileInput>({
		resolver: zodResolver(profileSchema),
		values: { name },
	})

	const update = useMutation({
		mutationKey: mutationKeys.auth.updateProfile,
		mutationFn: async (data: ProfileInput) => {
			const result = await authClient.updateUser({ name: data.name })
			if (result.error) throw new Error(result.error.message ?? "Update failed")
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: queryKeys.auth.me })
		},
	})

	return (
		<SettingsRow title="Profile" description="Your personal information visible to team members.">
			<form onSubmit={form.handleSubmit((d) => update.mutate(d))} className="space-y-4">
				{update.isSuccess && (
					<Alert variant="success">
						<CheckCircle />
						<AlertDescription>Profile updated.</AlertDescription>
					</Alert>
				)}
				{update.error && (
					<Alert variant="danger">
						<AlertCircle />
						<AlertDescription>{update.error.message}</AlertDescription>
					</Alert>
				)}

				<div className="space-y-1.5">
					<Label htmlFor="profile-name" className="text-xs">Name</Label>
					<Input id="profile-name" {...form.register("name")} feedback={form.formState.errors.name ? "error" : undefined} />
					{form.formState.errors.name && <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>}
				</div>

				<div className="space-y-1.5">
					<Label className="text-xs">Email</Label>
					<Input value={email} disabled />
					<p className="text-xs text-muted-foreground">Contact support to change your email.</p>
				</div>

				<Button type="submit" variant="solid" size="sm" isLoading={update.isPending} disabled={!form.formState.isDirty}>
					Save changes
				</Button>
			</form>
		</SettingsRow>
	)
}

// ── Password ──

function PasswordRow() {
	const form = useForm<PasswordFormInput>({
		resolver: zodResolver(passwordSchema),
		defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
	})

	const change = useMutation({
		mutationKey: mutationKeys.auth.changePassword,
		mutationFn: async (data: PasswordFormInput) => {
			const result = await authClient.changePassword({
				currentPassword: data.currentPassword,
				newPassword: data.newPassword,
			})
			if (result.error) throw new Error(result.error.message ?? "Password change failed")
		},
		onSuccess: () => form.reset(),
	})

	return (
		<SettingsRow title="Password" description="Change the password used to sign in to your account.">
			<form onSubmit={form.handleSubmit((d) => change.mutate(d))} className="space-y-4">
				{change.isSuccess && (
					<Alert variant="success">
						<CheckCircle />
						<AlertDescription>Password changed.</AlertDescription>
					</Alert>
				)}
				{change.error && (
					<Alert variant="danger">
						<AlertCircle />
						<AlertDescription>{change.error.message}</AlertDescription>
					</Alert>
				)}

				<div className="space-y-1.5">
					<Label htmlFor="current-password" className="text-xs">Current password</Label>
					<PasswordInput
						id="current-password"
						placeholder="Enter current password"
						{...form.register("currentPassword")}
						feedback={form.formState.errors.currentPassword ? "error" : undefined}
					/>
					{form.formState.errors.currentPassword && (
						<p className="text-xs text-destructive">{form.formState.errors.currentPassword.message}</p>
					)}
				</div>

				<div className="grid gap-4 sm:grid-cols-2">
					<div className="space-y-1.5">
						<Label htmlFor="new-password" className="text-xs">New password</Label>
						<PasswordInput
							id="new-password"
							placeholder="New password"
							{...form.register("newPassword")}
							feedback={form.formState.errors.newPassword ? "error" : undefined}
						/>
						{form.formState.errors.newPassword && (
							<p className="text-xs text-destructive">{form.formState.errors.newPassword.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="confirm-password" className="text-xs">Confirm password</Label>
						<PasswordInput
							id="confirm-password"
							placeholder="Confirm password"
							{...form.register("confirmPassword")}
							feedback={form.formState.errors.confirmPassword ? "error" : undefined}
						/>
						{form.formState.errors.confirmPassword && (
							<p className="text-xs text-destructive">{form.formState.errors.confirmPassword.message}</p>
						)}
					</div>
				</div>

				<Button type="submit" variant="solid" size="sm" isLoading={change.isPending}>
					Update password
				</Button>
			</form>
		</SettingsRow>
	)
}

// ── Linked Accounts ──

function LinkedAccountsRow() {
	const { data: accounts } = useQuery({
		queryKey: ["auth", "accounts"],
		queryFn: async () => {
			const result = await authClient.listAccounts()
			if (result.error) return []
			return result.data ?? []
		},
	})

	const link = useMutation({
		mutationKey: mutationKeys.auth.linkSocial,
		mutationFn: async (provider: "github" | "google") => {
			await authClient.linkSocial({ provider, callbackURL: window.location.href })
		},
	})

	const providers = [
		{ id: "github" as const, label: "GitHub", icon: Github },
		{ id: "google" as const, label: "Google", icon: Mail },
	]

	return (
		<SettingsRow title="Linked accounts" description="Connect third-party accounts for faster sign in.">
			<div className="space-y-3">
				{providers.map((provider) => {
					const linked = accounts?.some((a: { providerId?: string }) => a.providerId === provider.id)
					return (
						<div key={provider.id} className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
							<div className="flex items-center gap-2.5">
								<provider.icon className="size-4 text-muted-foreground" />
								<span className="text-sm">{provider.label}</span>
								{linked && <Badge variant="success">Connected</Badge>}
							</div>
							{!linked && (
								<Button variant="outline" size="xs" onClick={() => link.mutate(provider.id)} isLoading={link.isPending}>
									Connect
								</Button>
							)}
						</div>
					)
				})}
			</div>
		</SettingsRow>
	)
}

