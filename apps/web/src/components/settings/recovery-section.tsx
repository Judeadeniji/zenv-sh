import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { wrapWithPublicKey } from "@zenv/amnesia"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Badge } from "#/components/ui/badge"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Switch } from "#/components/ui/switch"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { fromBase64, toBase64 } from "#/lib/encoding"
import { useAuthStore } from "#/lib/stores/auth"
import { AlertCircle, EyeIcon, ShieldAlert, Users } from "lucide-react"
import { Link, useNavigate } from "@tanstack/react-router"

const addContactSchema = z.object({
	email: z.email("Enter a valid email address"),
})
type AddContactInput = z.infer<typeof addContactSchema>

interface RecoverySectionProps {
	action?: string
}

export function RecoverySection({ action }: RecoverySectionProps) {
	const { data: status, isLoading } = useQuery({
		queryKey: queryKeys.recovery.status,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/status")
			if (error || !data) throw new Error("Failed to fetch recovery status")
			return data;
		},
	})

	if (isLoading) {
		return (
			<div className="space-y-6 py-6">
				{[1, 2, 3].map((i) => (
					<div key={i} className="h-20 animate-pulse rounded-md bg-muted/50" />
				))}
			</div>
		)
	}

	return (
		<div>
			<RecoveryKitRow hasKit={status?.has_kit ?? false} />
			<SettingsDivider />
			<TrustedContactRow
				hasContact={status?.has_contact ?? false}
				contactEmail={status?.contact_email}
				showAddForm={action === "add-contact"}
			/>
			<SettingsDivider />
			<IncomingRequestsLinkRow />
			<SettingsDivider />
			<NoRecoveryRow disabled={status?.recovery_disabled ?? false} />
		</div>
	)
}

// ── Recovery Kit ──

function RecoveryKitRow({ hasKit }: { hasKit: boolean }) {
	return (
		<SettingsRow
			title="Recovery Kit"
			description="A 12-word mnemonic phrase that can recover your vault if you forget your Vault Key. Store it somewhere safe."
		>
			<div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
				<div className="flex items-center gap-2">
					<span className="text-sm">{hasKit ? "Kit configured" : "Not set up"}</span>
					{hasKit ? <Badge variant="success">Active</Badge> : <Badge variant="warning">Missing</Badge>}
				</div>
				<Button
					variant="outline"
					size="xs"
					render={
						<Link
							to={hasKit ? "/recover/kit" : "/vault-setup"}
							search={hasKit ? { regenerate: true } : undefined}
						/>
					}
				>
					{hasKit ? "Regenerate" : "Set up"}
				</Button>
			</div>
		</SettingsRow>
	)
}

// ── Trusted Contact ──

function TrustedContactRow({
	hasContact,
	contactEmail,
	showAddForm,
}: {
	hasContact: boolean
	contactEmail?: string
	showAddForm?: boolean
}) {
	const qc = useQueryClient()
	const navigate = useNavigate()
	const crypto = useAuthStore((s) => s.crypto)
	const [adding, setAdding] = useState(showAddForm && !hasContact)

	const remove = useMutation({
		mutationKey: mutationKeys.recovery.removeContact,
		mutationFn: async () => {
			const { error } = await api().DELETE("/auth/recovery/trusted-contact")
			if (error) throw new Error("Failed to remove trusted contact")
		},
		onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recovery.status }),
	})

	const addContact = useMutation({
		mutationKey: mutationKeys.recovery.setContact,
		mutationFn: async ({ email }: AddContactInput) => {
			if (!crypto) throw new Error("Vault must be unlocked to set a trusted contact")

			const { data: pkData, error: pkErr } = await api().GET("/users/public-key", {
				params: { query: { email } },
			})
			if (pkErr || !pkData) throw new Error("Failed to look up contact public key")

			const publicKey = fromBase64((pkData as { public_key: string }).public_key)
			const wrapped = wrapWithPublicKey(crypto.dek, publicKey)

			const { error } = await api().POST("/auth/recovery/trusted-contact", {
				body: { contact_email: email, trusted_wrapped_dek: toBase64(wrapped) },
			})
			if (error) throw new Error("Failed to add trusted contact")
		},
		onSuccess: async () => {
			await qc.invalidateQueries({ queryKey: queryKeys.recovery.status })
			await qc.invalidateQueries({ queryKey: queryKeys.recovery.incomingRequests })
			setAdding(false)
			navigate({ to: "/settings", search: { tab: "recovery" } })
		},
	})

	const form = useForm<AddContactInput>({
		resolver: zodResolver(addContactSchema),
		defaultValues: { email: "" },
	})

	return (
		<SettingsRow
			title="Trusted Contact"
			description="Designate someone to help recover your vault. A 72-hour waiting period applies before recovery completes."
		>
			{remove.error && (
				<Alert variant="danger" className="mb-3">
					<AlertCircle />
					<AlertDescription>{remove.error.message}</AlertDescription>
				</Alert>
			)}

			{adding ? (
				<form onSubmit={form.handleSubmit((d) => addContact.mutate(d))} className="space-y-4">
					{addContact.error && (
						<Alert variant="danger">
							<AlertCircle />
							<AlertDescription>{addContact.error.message}</AlertDescription>
						</Alert>
					)}

					<div className="space-y-1.5">
						<Label htmlFor="contact-email" className="text-xs">Contact's email</Label>
						<Input
							id="contact-email"
							type="email"
							placeholder="colleague@company.com"
							{...form.register("email")}
							feedback={form.formState.errors.email ? "error" : undefined}
						/>
						{form.formState.errors.email && (
							<p className="text-xs text-destructive">{form.formState.errors.email.message}</p>
						)}
						<p className="text-xs text-muted-foreground">
							This person must also have a zEnv account. They'll be notified when you request recovery.
						</p>
					</div>

					<div className="flex gap-2">
						<Button type="submit" variant="solid" size="sm" isLoading={addContact.isPending}>
							Add contact
						</Button>
						<Button type="button" variant="ghost" size="sm" onClick={() => setAdding(false)}>
							Cancel
						</Button>
					</div>
				</form>
			) : (
				<div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
					{hasContact ? (
						<>
							<div>
								<p className="text-sm font-medium">{contactEmail}</p>
								<p className="text-xs text-muted-foreground">Will be notified on recovery request.</p>
							</div>
							<Button variant="outline" size="xs" onClick={() => remove.mutate()} isLoading={remove.isPending}>
								Remove
							</Button>
						</>
					) : (
						<>
							<div className="flex items-center gap-2">
								<span className="text-sm">No contact configured</span>
								<Badge variant="neutral">Optional</Badge>
							</div>
							<Button variant="outline" size="xs" onClick={() => setAdding(true)}>
								Add
							</Button>
						</>
					)}
				</div>
			)}
		</SettingsRow>
	)
}

function IncomingRequestsLinkRow() {
	const { data } = useQuery({
		queryKey: queryKeys.recovery.incomingRequests,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/incoming-requests")
			if (error) return []
			return (data ?? [])
		},
		refetchInterval: 30_000,
	})

	const pendingCount = (data ?? []).filter((r) => r.status === "pending").length
	return (
		<SettingsRow
			title="Recovery Requests"
			description="Manage vault reset requests from users who have designated you as their trusted recovery contact."
		>
			<div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
				<div className="flex items-center gap-2">
					<Users className="size-4 text-muted-foreground" />
					<span className="text-sm">Incoming requests</span>
					{pendingCount > 0 && <Badge variant="warning">{pendingCount}</Badge>}
				</div>
				<Button
					variant="outline"
					size="xs"
					render={<Link to="/recovery-requests" />}
				>
					<EyeIcon />
				</Button>
			</div>
		</SettingsRow>
	)
}

// ── No Recovery ──

function NoRecoveryRow({ disabled }: { disabled: boolean }) {
	const qc = useQueryClient()

	const toggle = useMutation({
		mutationKey: mutationKeys.recovery.toggleDisable,
		mutationFn: async () => {
			const { error } = await api().PUT("/auth/recovery/disable", {
				body: { disabled: !disabled },
			})
			if (error) throw new Error("Failed to update recovery setting")
		},
		onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recovery.status }),
	})

	return (
		<SettingsRow
			title="Disable Recovery"
			description="For maximum security, disable all recovery methods. If you lose your Vault Key, your data is permanently gone."
		>
			{toggle.error && (
				<Alert variant="danger" className="mb-3">
					<AlertCircle />
					<AlertDescription>{toggle.error.message}</AlertDescription>
				</Alert>
			)}

			<div className="space-y-3">
				<div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
					<div>
						<p className="text-sm font-medium">{disabled ? "Recovery disabled" : "Recovery enabled"}</p>
						<p className="text-xs text-muted-foreground">
							{disabled ? "No recovery methods available." : "Kit and Trusted Contact can be used."}
						</p>
					</div>
					<Switch checked={disabled} onCheckedChange={() => toggle.mutate()} disabled={toggle.isPending} />
				</div>

				{disabled && (
					<Alert variant="danger">
						<ShieldAlert />
						<AlertDescription>
							If you forget your Vault Key, your secrets will be permanently lost.
						</AlertDescription>
					</Alert>
				)}
			</div>
		</SettingsRow>
	)
}
