import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Textarea } from "#/components/ui/textarea"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useCreateSecret } from "#/lib/queries/secrets"
import { useProjectDEK } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import { createSecretSchema, type CreateSecretInput } from "#/lib/schemas/secrets"
import { AlertCircle } from "lucide-react"

interface CreateSecretDialogProps {
	projectId: string
	trigger: React.ReactElement
}

export function CreateSecretDialog({ projectId, trigger }: CreateSecretDialogProps) {
	const [open, setOpen] = useState(false)
	const environment = useNavStore((s) => s.activeEnvironment)
	const { data: projectDEK } = useProjectDEK(projectId)
	const create = useCreateSecret()

	const form = useForm<CreateSecretInput>({
		resolver: zodResolver(createSecretSchema),
		defaultValues: { name: "", value: "" },
	})

	const onSubmit = (data: CreateSecretInput) => {
		if (!projectDEK) return
		create.mutate(
			{ projectId, environment, projectDEK, ...data },
			{
				onSuccess: () => {
					setOpen(false)
					form.reset()
				},
			},
		)
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { setOpen(v); if (!v) form.reset() }}>
			<DialogTrigger render={trigger} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add a secret</DialogTitle>
					<DialogDescription>
						The value is encrypted on your device before being sent to the server.
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
						<Label htmlFor="secret-name" className="text-xs">Name</Label>
						<Input
							id="secret-name"
							placeholder="e.g. api-key, db/password, MY_SECRET"
							{...form.register("name")}
							feedback={form.formState.errors.name ? "error" : undefined}
							autoFocus
						/>
						{form.formState.errors.name && (
							<p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="secret-value" className="text-xs">Value</Label>
						<Textarea
							id="secret-value"
							placeholder="The value to encrypt"
							className="font-mono text-xs"
							rows={3}
							{...form.register("value")}
							feedback={form.formState.errors.value ? "error" : undefined}
						/>
						{form.formState.errors.value && (
							<p className="text-xs text-destructive">{form.formState.errors.value.message}</p>
						)}
					</div>

					<DialogFooter>
						<DialogClose>
							<Button variant="ghost" size="sm" type="button">Cancel</Button>
						</DialogClose>
						<Button type="submit" variant="solid" size="sm" isLoading={create.isPending}>
							Add secret
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	)
}
