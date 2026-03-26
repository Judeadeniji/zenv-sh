import { useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "#/components/ui/select"
import { useAddMember } from "#/lib/queries/orgs"
import { inviteMemberSchema, type InviteMemberInput } from "#/lib/schemas/secrets"
import { AlertCircle, CheckCircle } from "lucide-react"

interface InviteMemberDialogProps {
	orgId: string
	trigger: React.ReactElement
}

export function InviteMemberDialog({ orgId, trigger }: InviteMemberDialogProps) {
	const [open, setOpen] = useState(false)
	const [success, setSuccess] = useState(false)
	const addMember = useAddMember()

	const form = useForm({
		resolver: zodResolver(inviteMemberSchema),
		defaultValues: { email: "", role: "dev" },
	})

	const onSubmit = (data: InviteMemberInput) => {
		addMember.mutate(
			{ orgId, email: data.email, role: data.role },
			{
				onSuccess: () => {
					setSuccess(true)
					form.reset()
				},
			},
		)
	}

	const handleClose = () => {
		setOpen(false)
		setSuccess(false)
		form.reset()
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); else setOpen(true) }}>
			<DialogTrigger render={trigger} nativeButton={false} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Invite a member</DialogTitle>
					<DialogDescription>
						They'll receive an email invitation to join your organization.
					</DialogDescription>
				</DialogHeader>

				<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
					{addMember.error && (
						<Alert variant="danger">
							<AlertCircle />
							<AlertDescription>{addMember.error.message}</AlertDescription>
						</Alert>
					)}

					{success && (
						<Alert variant="success">
							<CheckCircle />
							<AlertDescription>Invitation sent! You can invite another member or close this dialog.</AlertDescription>
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
							<p className="text-xs te blockxt-destructive">{form.formState.errors.email.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label className="text-xs block">Role</Label>
						<Controller
							control={form.control}
							name="role"
							render={({ field }) => (
								<Select value={field.value} onValueChange={field.onChange}>
									<SelectTrigger className={"w-full"}>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="admin">Admin</SelectItem>
										<SelectItem value="senior_dev">Senior Dev</SelectItem>
										<SelectItem value="dev">Dev</SelectItem>
										<SelectItem value="contractor">Contractor</SelectItem>
										<SelectItem value="ci_bot">CI Bot</SelectItem>
									</SelectContent>
								</Select>
							)}
						/>
					</div>

					<DialogFooter>
						<DialogClose>
							<Button variant="ghost" size="sm" type="button">Cancel</Button>
						</DialogClose>
						<Button type="submit" variant="solid" size="sm" isLoading={addMember.isPending}>
							Send invite
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	)
}
