import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "#/components/ui/select"
import { authClient } from "#/lib/auth-client"
import { AlertCircle, Copy, Check, Link } from "lucide-react"

const inviteSchema = z.object({
	email: z.string().email("Enter a valid email address"),
	role: z.enum(["admin", "member", "owner"]).default("member"),
})
type InviteInput = z.infer<typeof inviteSchema>

interface InviteMemberDialogProps {
	orgId: string
	trigger: React.ReactElement
}

export function InviteMemberDialog({ orgId, trigger }: InviteMemberDialogProps) {
	const [open, setOpen] = useState(false)
	const [inviteLink, setInviteLink] = useState<string | null>(null)
	const [copied, setCopied] = useState(false)
	const [isPending, setIsPending] = useState(false)
	const [error, setError] = useState<string | null>(null)

	const form = useForm<InviteInput>({
		resolver: zodResolver(inviteSchema),
		defaultValues: { email: "", role: "member" },
	})

	const handleClose = () => {
		setOpen(false)
		setInviteLink(null)
		setCopied(false)
		setError(null)
		form.reset()
	}

	const onSubmit = async (data: InviteInput) => {
		setError(null)
		setIsPending(true)
		try {
			const result = await authClient.organization.inviteMember({
				email: data.email,
				role: data.role,
				organizationId: orgId,
			})
			if (result.error) {
				setError(result.error.message ?? "Failed to create invitation")
				return
			}
			const invitationId = result.data?.id
			if (!invitationId) {
				setError("No invitation ID returned")
				return
			}
			setInviteLink(`${window.location.origin}/join/${invitationId}`)
			form.reset()
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to create invitation")
		} finally {
			setIsPending(false)
		}
	}

	const handleCopy = () => {
		if (!inviteLink) return
		navigator.clipboard?.writeText(inviteLink).then(() => {
			setCopied(true)
			setTimeout(() => setCopied(false), 2000)
		})
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); else setOpen(true) }}>
			<DialogTrigger render={trigger} nativeButton={false} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Invite a member</DialogTitle>
					<DialogDescription>
						Generate a one-time invite link. When they visit it they'll be added to your organization.
					</DialogDescription>
				</DialogHeader>

				{inviteLink ? (
					<div className="space-y-4 py-2">
						<div className="rounded-lg border border-border bg-muted/40 p-4 text-center">
							<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
								<Link className="size-4 text-primary" />
							</div>
							<p className="text-sm font-medium">Invite link created</p>
							<p className="mt-0.5 text-xs text-muted-foreground">
								Share this link with the invitee. It expires in 48 hours.
							</p>
						</div>

						<div className="flex items-center gap-2">
							<code className="flex-1 truncate rounded-md border border-border bg-muted px-3 py-2 font-mono text-xs">
								{inviteLink}
							</code>
							<Button variant="outline" size="icon-sm" onClick={handleCopy} title="Copy invite link">
								{copied ? <Check className="size-3.5 text-success" /> : <Copy className="size-3.5" />}
							</Button>
						</div>

						<p className="text-[11px] text-muted-foreground">
							Email sending isn't configured yet — share this link manually.
						</p>

						<DialogFooter>
							<Button variant="ghost" size="sm" type="button" onClick={() => { setInviteLink(null); setError(null) }}>
								Invite another
							</Button>
							<DialogClose>
								<Button variant="solid" size="sm" type="button">Done</Button>
							</DialogClose>
						</DialogFooter>
					</div>
				) : (
					<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
						{error && (
							<Alert variant="danger">
								<AlertCircle />
								<AlertDescription>{error}</AlertDescription>
							</Alert>
						)}

						<div className="space-y-1.5">
							<Label htmlFor="invite-email" className="text-xs block">Email address</Label>
							<Input
								id="invite-email"
								type="email"
								placeholder="teammate@company.com"
								{...form.register("email")}
								feedback={form.formState.errors.email ? "error" : undefined}
								autoFocus
							/>
							{form.formState.errors.email && (
								<p className="text-xs text-destructive">{form.formState.errors.email.message}</p>
							)}
						</div>

						<div className="space-y-1.5">
							<Label className="text-xs block">Role</Label>
							<Select
								value={form.watch("role")}
								onValueChange={(v) => form.setValue("role", v as InviteInput["role"])}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="owner">Owner</SelectItem>
									<SelectItem value="admin">Admin</SelectItem>
									<SelectItem value="member">Member</SelectItem>
								</SelectContent>
							</Select>
						</div>

						<DialogFooter>
							<DialogClose>
								<Button variant="ghost" size="sm" type="button">Cancel</Button>
							</DialogClose>
							<Button type="submit" variant="solid" size="sm" isLoading={isPending}>
								Generate invite link
							</Button>
						</DialogFooter>
					</form>
				)}
			</DialogContent>
		</Dialog>
	)
}
