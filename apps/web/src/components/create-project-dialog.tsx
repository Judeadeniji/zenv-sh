import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useCreateProject } from "#/lib/queries/projects"
import { createProjectSchema, type CreateProjectInput } from "#/lib/schemas/onboarding"
import { AlertCircle } from "lucide-react"

interface CreateProjectDialogProps {
	orgId: string
	trigger: React.ReactElement
}

export function CreateProjectDialog({ orgId, trigger }: CreateProjectDialogProps) {
	const [open, setOpen] = useState(false)
	const navigate = useNavigate()
	const create = useCreateProject()

	const form = useForm<CreateProjectInput>({
		resolver: zodResolver(createProjectSchema),
		defaultValues: { name: "" },
	})

	const onSubmit = (data: CreateProjectInput) => {
		create.mutate(
			{ name: data.name, orgId },
			{
				onSuccess: (result) => {
					setOpen(false)
					form.reset()
					navigate({
						to: "/orgs/$orgId/projects/$projectId/secrets",
						params: { orgId, projectId: result.id ?? "" },
					})
				},
			},
		)
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { setOpen(v); if (!v) { form.reset(); create.reset() } }}>
			<DialogTrigger render={trigger} nativeButton={false} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>New project</DialogTitle>
					<DialogDescription>
						Projects contain your secrets, organized by environment.
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
						<Label htmlFor="project-name" className="text-xs">Project name</Label>
						<Input
							id="project-name"
							placeholder="my-app"
							{...form.register("name")}
							feedback={form.formState.errors.name ? "error" : undefined}
							autoFocus
						/>
						{form.formState.errors.name && (
							<p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
						)}
						<p className="text-xs text-muted-foreground">Lowercase letters, numbers, and hyphens.</p>
					</div>

					<DialogFooter>
						<DialogClose>
							<Button variant="ghost" size="sm" type="button">Cancel</Button>
						</DialogClose>
						<Button type="submit" variant="solid" size="sm" isLoading={create.isPending}>
							Create project
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	)
}
