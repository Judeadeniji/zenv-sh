import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
	DialogFooter,
	DialogClose,
} from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Textarea } from "#/components/ui/textarea"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Spinner } from "#/components/ui/spinner"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "#/components/ui/collapsible"
import { useUpdateSecret } from "#/lib/queries/secrets"
import type { DecryptedSecretRow } from "#/lib/queries/secrets"
import { useProjectDEK } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import {
	updateSecretSchema,
	type UpdateSecretInput,
	buildSecretMetadataPayload,
} from "#/lib/schemas/secrets"
import { inferMimeForUpdateClient } from "#/lib/secret-mime"
import { toast } from "sonner"
import { AlertCircle, ChevronDown } from "lucide-react"
import { cn } from "#/lib/utils"

interface EditSecretDialogProps {
	projectId: string
	secret: DecryptedSecretRow | null
	open: boolean
	onOpenChange: (open: boolean) => void
	/** Waiting for lazy ciphertext fetch / decrypt */
	isLoading?: boolean
}

export function EditSecretDialog({
	projectId,
	secret,
	open,
	onOpenChange,
	isLoading = false,
}: EditSecretDialogProps) {
	const environment = useNavStore((s) => s.activeEnvironment)
	const { data: projectDEK } = useProjectDEK(projectId)
	const update = useUpdateSecret()
	const [metaOpen, setMetaOpen] = useState(false)

	const meta = secret?.metadata
	const form = useForm<UpdateSecretInput>({
		resolver: zodResolver(updateSecretSchema),
		values: {
			value: secret?.value ?? "",
			description: meta?.description ?? "",
			tags_input: meta?.tags?.join(", ") ?? "",
		},
	})

	const onSubmit = async (data: UpdateSecretInput) => {
		if (!projectDEK || !secret || secret.kind === "binary") return
		const mime = await inferMimeForUpdateClient(secret, data.value)
		const metadata = buildSecretMetadataPayload(
			{
				description: data.description,
				tags_input: data.tags_input,
			},
			mime,
		)
		update.mutate(
			{
				projectId,
				environment,
				nameHash: secret.name_hash,
				name: secret.name,
				value: data.value,
				projectDEK,
				metadata,
			},
			{
				onSuccess: () => {
					onOpenChange(false)
					toast.success(`Updated ${secret.name}`)
				},
				onError: (err) => toast.error(err.message || "Failed to update secret"),
			},
		)
	}

	const showForm = Boolean(secret && secret.kind !== "binary" && !isLoading)

	return (
		<Dialog
			open={open}
			onOpenChange={(v) => {
				onOpenChange(v)
				if (!v) {
					form.reset()
					update.reset()
					setMetaOpen(false)
				}
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Edit secret</DialogTitle>
					<DialogDescription>
						Values are re-encrypted locally. Optional details below are merged as plaintext metadata
						on the server.
					</DialogDescription>
				</DialogHeader>

				{isLoading && !secret ? (
					<div className="flex justify-center py-12">
						<Spinner />
					</div>
				) : null}

				{secret?.kind === "binary" ? (
					<div className="grid gap-4 py-2">
						<p className="text-sm text-muted-foreground">
							File secrets cannot be edited as text in the dashboard.
						</p>
						<DialogFooter>
							<Button type="button" variant="solid" size="sm" onClick={() => onOpenChange(false)}>
								Close
							</Button>
						</DialogFooter>
					</div>
				) : null}

				{!isLoading && !secret ? (
					<p className="py-6 text-center text-sm text-muted-foreground">
						Could not load this secret.
					</p>
				) : null}

				{showForm && secret ? (
					<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
						{update.error && (
							<Alert variant="danger">
								<AlertCircle />
								<AlertDescription>{update.error.message}</AlertDescription>
							</Alert>
						)}

						<div className="space-y-1.5">
							<Label htmlFor="edit-secret-name" className="text-xs">
								Name
							</Label>
							<code className="block rounded bg-muted px-2 py-1.5 font-mono text-xs font-semibold text-muted-foreground">
								{secret.name}
							</code>
						</div>

						<div className="space-y-1.5">
							<Label htmlFor="edit-secret-value" className="text-xs">
								Value
							</Label>
							<Textarea
								id="edit-secret-value"
								placeholder="The new value to encrypt"
								className="font-mono text-xs"
								rows={3}
								{...form.register("value")}
								feedback={form.formState.errors.value ? "error" : undefined}
								autoFocus
							/>
							{form.formState.errors.value && (
								<p className="text-xs text-destructive">{form.formState.errors.value.message}</p>
							)}
						</div>

						<Collapsible
							open={metaOpen}
							onOpenChange={setMetaOpen}
							className="rounded-md border bg-muted/20"
						>
							<CollapsibleTrigger
								type="button"
								className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs font-medium text-muted-foreground hover:text-foreground"
							>
								<span>Server-visible details</span>
								<ChevronDown
									className={cn("size-4 shrink-0 transition-transform", metaOpen && "rotate-180")}
								/>
							</CollapsibleTrigger>
							<CollapsibleContent className="border-t px-3 pb-3 pt-1">
								<div className="grid gap-3">
									<div className="space-y-1">
										<Label htmlFor="edit-desc" className="text-[11px] text-muted-foreground">
											Description
										</Label>
										<Textarea
											id="edit-desc"
											className="min-h-[64px] text-xs"
											rows={2}
											{...form.register("description")}
										/>
									</div>
									<div className="space-y-1">
										<Label htmlFor="edit-tags" className="text-[11px] text-muted-foreground">
											Tags (comma-separated)
										</Label>
										<Input
											id="edit-tags"
											className="h-8 text-xs"
											{...form.register("tags_input")}
										/>
									</div>
								</div>
							</CollapsibleContent>
						</Collapsible>

						<DialogFooter>
							<DialogClose>
								<Button variant="ghost" size="sm" type="button">
									Cancel
								</Button>
							</DialogClose>
							<Button type="submit" variant="solid" size="sm" isLoading={update.isPending}>
								Save changes
							</Button>
						</DialogFooter>
					</form>
				) : null}
			</DialogContent>
		</Dialog>
	)
}
