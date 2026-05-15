import { useState, useRef, useCallback } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import {
	Dialog, DialogTrigger, DialogContent, DialogHeader,
	DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Textarea } from "#/components/ui/textarea"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useCreateSecret } from "#/lib/queries/secrets"
import { useProjectDEK } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import { createSecretSchema, type CreateSecretInput, buildSecretMetadataPayload } from "#/lib/schemas/secrets"
import { toast } from "sonner"
import { AlertCircle, File, Upload, X, ChevronDown } from "lucide-react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "#/components/ui/collapsible"
import { cn } from "#/lib/utils"

type InputMode = "text" | "file"

interface CreateSecretDialogProps {
	projectId: string
	trigger: React.ReactElement
}

function formatBytes(bytes: number) {
	if (bytes < 1024) return `${bytes} B`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function CreateSecretDialog({ projectId, trigger }: CreateSecretDialogProps) {
	const [open, setOpen] = useState(false)
	const [inputMode, setInputMode] = useState<InputMode>("text")
	const [file, setFile] = useState<File | null>(null)
	const [fileError, setFileError] = useState<string | null>(null)
	const [isDragging, setIsDragging] = useState(false)
	const [metaOpen, setMetaOpen] = useState(false)
	const fileInputRef = useRef<HTMLInputElement>(null)

	const environment = useNavStore((s) => s.activeEnvironment)
	const { data: projectDEK } = useProjectDEK(projectId)
	const create = useCreateSecret()

	const form = useForm<CreateSecretInput>({
		resolver: zodResolver(createSecretSchema),
		defaultValues: { inputMode: "text", name: "", value: "", mime_type: "", description: "", tags_input: "" },
	})


	const resetDialog = useCallback(() => {
		form.reset()
		setInputMode("text")
		setFile(null)
		setFileError(null)
		setIsDragging(false)
		setMetaOpen(false)
	}, [form])
	
	const handleModeSwitch = (mode: InputMode) => {
		setInputMode(mode)
		setFile(null)
		setFileError(null)
		form.setValue("inputMode", mode) // keep RHF in sync
		form.clearErrors()
	}
	
	const acceptFile = (incoming: File) => {
		if (incoming.size > 1_048_576) {
			setFileError("File exceeds the 1 MB limit")
			return
		}
		setFile(incoming)
		setFileError(null)
	}
	
	const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
		const f = e.target.files?.[0]
		if (f) acceptFile(f)
		// reset so the same file can be re-selected after clearing
		e.target.value = ""
	}

	const handleDrop = (e: React.DragEvent) => {
		e.preventDefault()
		setIsDragging(false)
		const f = e.dataTransfer.files[0]
		if (f) acceptFile(f)
	}

	const onSubmit = async (data: CreateSecretInput) => {
		if (!projectDEK) { toast.error("No project DEK found"); return }
	
		const metadata = buildSecretMetadataPayload({
			mime_type: data.mime_type,
			description: data.description,
			tags_input: data.tags_input,
		})

		if (data.inputMode === "file") {
			if (!file) { setFileError("Select a file to encrypt"); return }
			const valueBytes = new Uint8Array(await file.arrayBuffer())
			create.mutate(
				{ projectId, environment, projectDEK, name: data.name, value: valueBytes, metadata },
				{
					onSuccess: () => { setOpen(false); resetDialog(); toast.success(`Created ${data.name}`) },
					onError: (err) => toast.error(err.message || "Failed to create secret"),
				},
			)
			return
		}
	
		create.mutate(
			{ projectId, environment, projectDEK, name: data.name, value: data.value, metadata },
			{
				onSuccess: () => { setOpen(false); resetDialog(); toast.success(`Created ${data.name}`) },
				onError: (err) => toast.error(err.message || "Failed to create secret"),
			},
		)
	}

	return (
		<Dialog open={open} onOpenChange={(v) => { setOpen(v); if (!v) resetDialog() }}>
			<DialogTrigger render={trigger} nativeButton={false} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add a secret</DialogTitle>
					<DialogDescription>
						The value is encrypted on your device before being sent to the server.
						Optional details below are stored in plaintext for search and tooling — never put secrets there.
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
						<div className="flex items-center justify-between">
							<Label className="text-xs">Value</Label>
							<div className="flex rounded-md border text-xs overflow-hidden">
								{(["text", "file"] as InputMode[]).map((mode) => (
									<button
										key={mode}
										type="button"
										onClick={() => handleModeSwitch(mode)}
										className={cn(
											"px-2.5 py-1 capitalize transition-colors",
											inputMode === mode
												? "bg-foreground text-background"
												: "text-muted-foreground hover:text-foreground",
										)}
									>
										{mode}
									</button>
								))}
							</div>
						</div>

						{inputMode === "text" ? (
							<>
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
							</>
						) : (
							<>
								{file ? (
									<div className="flex items-center gap-2.5 rounded-md border bg-muted/40 px-3 py-2.5">
										<File className="size-4 shrink-0 text-muted-foreground" />
										<div className="min-w-0 flex-1">
											<p className="truncate text-xs font-medium">{file.name}</p>
											<p className="text-xs text-muted-foreground">
												{formatBytes(file.size)}{file.type ? ` · ${file.type}` : ""}
											</p>
										</div>
										<Button
											type="button"
											variant="ghost"
											size="icon"
											className="size-6 shrink-0"
											onClick={() => { setFile(null); setFileError(null) }}
											aria-label="Remove file"
										>
											<X className="size-3.5" />
										</Button>
									</div>
								) : (
									<button
										type="button"
										className={cn(
											"flex w-full flex-col items-center gap-1.5 rounded-md border border-dashed px-4 py-5 text-center transition-colors",
											isDragging
												? "border-foreground/40 bg-muted/60"
												: "border-border hover:border-foreground/30 hover:bg-muted/30",
											fileError && "border-destructive/60",
										)}
										onClick={() => fileInputRef.current?.click()}
										onDragOver={(e) => { e.preventDefault(); setIsDragging(true) }}
										onDragLeave={() => setIsDragging(false)}
										onDrop={handleDrop}
									>
										<Upload className="size-4 text-muted-foreground" />
										<span className="text-xs text-muted-foreground">
											Drop a file or{" "}
											<span className="text-foreground underline underline-offset-2">browse</span>
										</span>
										<span className="text-xs text-muted-foreground/60">
											Any file convertible to bytes
										</span>
									</button>
								)}

								{fileError && (
									<p className="text-xs text-destructive">{fileError}</p>
								)}

								<input
									ref={fileInputRef}
									type="file"
									className="hidden"
									onChange={handleFileInput}
								/>
							</>
						)}
					</div>

					<Collapsible open={metaOpen} onOpenChange={setMetaOpen} className="rounded-md border bg-muted/20">
						<CollapsibleTrigger
							type="button"
							className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs font-medium text-muted-foreground hover:text-foreground"
						>
							<span>Server-visible details (optional)</span>
							<ChevronDown className={cn("size-4 shrink-0 transition-transform", metaOpen && "rotate-180")} />
						</CollapsibleTrigger>
						<CollapsibleContent className="border-t px-3 pb-3 pt-1">
							<div className="grid gap-3">
								<div className="space-y-1">
									<Label htmlFor="secret-mime" className="text-[11px] text-muted-foreground">MIME type hint</Label>
									<Input
										id="secret-mime"
										placeholder="e.g. application/json, text/plain"
										className="h-8 text-xs"
										{...form.register("mime_type")}
									/>
								</div>
								<div className="space-y-1">
									<Label htmlFor="secret-desc" className="text-[11px] text-muted-foreground">Description</Label>
									<Textarea
										id="secret-desc"
										placeholder="What this secret is for (visible to zEnv operators)"
										className="min-h-[72px] text-xs"
										rows={3}
										{...form.register("description")}
									/>
								</div>
								<div className="space-y-1">
									<Label htmlFor="secret-tags" className="text-[11px] text-muted-foreground">Tags</Label>
									<Input
										id="secret-tags"
										placeholder="Comma-separated, e.g. prod, database, rotation"
										className="h-8 text-xs"
										{...form.register("tags_input")}
									/>
								</div>
							</div>
						</CollapsibleContent>
					</Collapsible>

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