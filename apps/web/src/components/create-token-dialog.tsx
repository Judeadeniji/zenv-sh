import { useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "#/components/ui/select"
import { OneTimeDisplay } from "#/components/ui/one-time-display"
import { useCreateToken } from "#/lib/queries/tokens"
import { useNavStore } from "#/lib/stores/nav"
import { createTokenSchema, type CreateTokenInput } from "#/lib/schemas/secrets"
import { AlertCircle } from "lucide-react"

interface CreateTokenDialogProps {
	projectId: string
	trigger: React.ReactElement
}

export function CreateTokenDialog({ projectId, trigger }: CreateTokenDialogProps) {
	const environment = useNavStore((s) => s.activeEnvironment)
	const [open, setOpen] = useState(false)
	const [createdToken, setCreatedToken] = useState<string | null>(null)
	const create = useCreateToken()

	const form = useForm<CreateTokenInput>({
		resolver: zodResolver(createTokenSchema),
		defaultValues: { name: "", permission: "read" },
	})

	const onSubmit = (data: CreateTokenInput) => {
		create.mutate(
			{ projectId, environment, ...data },
			{
				onSuccess: (res) => {
					const token = (res as { token?: string })?.token
					if (token) {
						setCreatedToken(token)
					} else {
						setOpen(false)
						form.reset()
					}
				},
			},
		)
	}

	const handleClose = () => {
		setOpen(false)
		setCreatedToken(null)
		form.reset()
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); else setOpen(true) }}>
			<DialogTrigger render={trigger} />
			<DialogContent>
				{createdToken ? (
					<>
						<DialogHeader>
							<DialogTitle>Token created</DialogTitle>
							<DialogDescription>
								Copy this token now — you won't be able to see it again.
							</DialogDescription>
						</DialogHeader>
						<div className="py-4">
							<OneTimeDisplay value={createdToken} label="Service Token" />
						</div>
						<DialogFooter>
							<Button variant="solid" size="sm" onClick={handleClose}>
								Done
							</Button>
						</DialogFooter>
					</>
				) : (
					<>
						<DialogHeader>
							<DialogTitle>Create a service token</DialogTitle>
							<DialogDescription>
								Tokens give your applications programmatic access to secrets.
							</DialogDescription>
						</DialogHeader>

						<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
							{create.error && (
								<Alert variant="danger">
									<AlertCircle />
									<AlertDescription>{create.error.message}</AlertDescription>
								</Alert>
							)}

							<div className="space-y-1.5">
								<Label htmlFor="token-name" className="text-xs">Name</Label>
								<Input
									id="token-name"
									placeholder="ci-deploy"
									{...form.register("name")}
									feedback={form.formState.errors.name ? "error" : undefined}
									autoFocus
								/>
								{form.formState.errors.name && (
									<p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
								)}
							</div>

							<div className="space-y-1.5">
								<Label className="text-xs">Permission</Label>
								<Controller
									control={form.control}
									name="permission"
									render={({ field }) => (
										<Select value={field.value} onValueChange={field.onChange}>
											<SelectTrigger>
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												<SelectItem value="read">Read only</SelectItem>
												<SelectItem value="read_write">Read &amp; write</SelectItem>
											</SelectContent>
										</Select>
									)}
								/>
							</div>

							<DialogFooter>
								<DialogClose>
									<Button variant="ghost" size="sm" type="button">Cancel</Button>
								</DialogClose>
								<Button type="submit" variant="solid" size="sm" isLoading={create.isPending}>
									Create token
								</Button>
							</DialogFooter>
						</form>
					</>
				)}
			</DialogContent>
		</Dialog>
	)
}
