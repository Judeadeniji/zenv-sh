import { useState, useMemo } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "#/components/ui/alert-dialog"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Input } from "#/components/ui/input"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Separator } from "#/components/ui/separator"
import { DataTable } from "#/components/data-table"
import { SearchInput } from "#/components/search-input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "#/components/ui/select"
import { CreateSecretDialog } from "#/components/create-secret-dialog"
import { ImportSecretsDialog } from "#/components/import-secrets-dialog"
import { EditSecretDialog } from "#/components/edit-secret-dialog"
import { useDecryptedSecrets, useDeleteSecret, useSecretVersions, useRollbackSecret } from "#/lib/queries/secrets"
import { useNavStore } from "#/lib/stores/nav"
import { toast } from "sonner"
import { KeyRound, Plus, Upload, Eye, EyeOff, Trash2, Copy, Check, Pencil, History, RotateCcw, AlertCircle } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/secrets")({
	component: SecretsPage,
})

interface DecryptedSecret {
	name_hash: string
	name: string
	value: string
	version?: number
	updated_at?: string
}

function SecretsPage() {
	const { projectId } = Route.useParams()
	const environment = useNavStore((s) => s.activeEnvironment)
	const { data: secrets, isLoading } = useDecryptedSecrets(projectId, environment)
	const [selectedSecret, setSelectedSecret] = useState<DecryptedSecret | null>(null)
	const [editingSecret, setEditingSecret] = useState<DecryptedSecret | null>(null)
	const [searchTerm, setSearchTerm] = useState("")
	const [sortBy, setSortBy] = useState<"name" | "updated">("name")
	const [sortDir, setSortDir] = useState<"asc" | "desc">("asc")
	const [versionFilter, setVersionFilter] = useState<"all" | "multi">("all")

	const allRows = secrets as DecryptedSecret[] ?? []
	const rows = useMemo(() => {
		let filtered = allRows
		if (searchTerm) {
			const q = searchTerm.toLowerCase()
			filtered = filtered.filter((s) => s.name.toLowerCase().includes(q))
		}
		if (versionFilter === "multi") {
			filtered = filtered.filter((s) => (s.version ?? 1) > 1)
		}
		return [...filtered].sort((a, b) => {
			if (sortBy === "name") {
				const cmp = a.name.localeCompare(b.name)
				return sortDir === "asc" ? cmp : -cmp
			}
			const aTime = a.updated_at ? new Date(a.updated_at).getTime() : 0
			const bTime = b.updated_at ? new Date(b.updated_at).getTime() : 0
			return sortDir === "asc" ? aTime - bTime : bTime - aTime
		})
	}, [allRows, searchTerm, sortBy, sortDir, versionFilter])

	const columns: ColumnDef<DecryptedSecret, unknown>[] = [
		{
			accessorKey: "name",
			header: "Name",
			cell: ({ row }) => (
				<code className="rounded bg-muted px-1.5 py-0.5 text-xs font-semibold">
					{row.original.name}
				</code>
			),
		},
		{
			id: "value",
			header: "Value",
			cell: ({ row }) => <MaskedValue value={row.original.value} />,
		},
		{
			accessorKey: "version",
			header: "Version",
			cell: ({ row }) => (
				<Badge variant="neutral" className="text-[10px]">v{row.original.version ?? 1}</Badge>
			),
		},
		{
			accessorKey: "updated_at",
			header: "Updated",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{row.original.updated_at ? new Date(row.original.updated_at).toLocaleDateString() : "—"}
				</span>
			),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) => (
				<div className="flex items-center justify-end gap-0.5">
					<Button
						variant="ghost"
						size="icon-sm"
						className="text-muted-foreground hover:text-foreground"
						type="button"
						onClick={(e) => {
							e.stopPropagation()
							setEditingSecret(row.original)
						}}
					>
						<Pencil />
					</Button>
					<DeleteSecretButton
						projectId={projectId}
						environment={environment}
						secret={row.original}
					/>
				</div>
			),
		},
	]

	if (isLoading) {
		return (
			<div>
				<div className="mb-6 flex items-center justify-between">
					<PageHeader />
				</div>
				<div className="flex items-center justify-center py-20"><Spinner /></div>
			</div>
		)
	}

	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<PageHeader />
				<div className="flex gap-2">
					<ImportSecretsDialog
						projectId={projectId}
						trigger={<Button type="button" variant="outline" size="sm"><Upload /> Import</Button>}
					/>
					<CreateSecretDialog
						projectId={projectId}
						trigger={<Button type="button" size="sm"><Plus /> Add</Button>}
					/>
				</div>
			</div>

			<div className="mb-4 flex items-center gap-3">
				<SearchInput
					placeholder="Search secrets..."
					value={searchTerm}
					onChange={setSearchTerm}
					debounceMs={150}
				/>
				<Select
					value={`${sortBy}-${sortDir}`}
					onValueChange={(val) => {
						if (!val) return
						const [by, dir] = val.split("-") as ["name" | "updated", "asc" | "desc"]
						setSortBy(by)
						setSortDir(dir)
					}}
				>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="Sort" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="name-asc">Name A→Z</SelectItem>
						<SelectItem value="name-desc">Name Z→A</SelectItem>
						<SelectItem value="updated-desc">Newest first</SelectItem>
						<SelectItem value="updated-asc">Oldest first</SelectItem>
					</SelectContent>
				</Select>
				<Select
					value={versionFilter}
					onValueChange={(val) => setVersionFilter(val as "all" | "multi")}
				>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="All versions" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All versions</SelectItem>
						<SelectItem value="multi">Multi-version</SelectItem>
					</SelectContent>
				</Select>
			</div>

			<DataTable
				columns={columns}
				data={rows}
				onRowClick={(row) => setSelectedSecret(row.original)}
				emptyIcon={<KeyRound />}
				emptyTitle="No secrets yet"
				emptyDescription="Secrets are encrypted on your device before leaving the browser. Add your first secret to get started."
				emptyAction={
					<div className="flex gap-2">
						<CreateSecretDialog
							projectId={projectId}
							trigger={<Button type="button" size="sm"><Plus /> Add a secret</Button>}
						/>
						<ImportSecretsDialog
							projectId={projectId}
							trigger={<Button type="button" variant="outline" size="sm"><Upload /> Import</Button>}
						/>
					</div>
				}
			/>

			<Sheet open={!!selectedSecret} onOpenChange={(open) => { if (!open) setSelectedSecret(null) }}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>{selectedSecret?.name ?? "Secret"}</SheetTitle>
						<SheetDescription>Decrypted in your browser. Never sent to the server.</SheetDescription>
					</SheetHeader>
					{selectedSecret && (
						<SecretDetailSheet
							projectId={projectId}
							environment={environment}
							secret={selectedSecret}
							onEdit={() => {
								setEditingSecret(selectedSecret)
								setSelectedSecret(null)
							}}
							onDeleted={() => setSelectedSecret(null)}
						/>
					)}
				</SheetContent>
			</Sheet>

			{editingSecret && (
				<EditSecretDialog
					projectId={projectId}
					secret={editingSecret}
					open={!!editingSecret}
					onOpenChange={(open) => { if (!open) setEditingSecret(null) }}
				/>
			)}
		</div>
	)
}

function SecretDetailSheet({ projectId, environment, secret, onEdit, onDeleted }: {
	projectId: string
	environment: string
	secret: DecryptedSecret
	onEdit: () => void
	onDeleted: () => void
}) {
	const [copied, setCopied] = useState<string | null>(null)
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const deleteSecret = useDeleteSecret()

	const handleCopy = (text: string, key: string) => {
		navigator.clipboard?.writeText(text).then(() => {
			setCopied(key)
			setTimeout(() => setCopied(null), 2000)
		})
	}

	const handleDelete = () => {
		deleteSecret.mutate(
			{ projectId, environment, nameHash: secret.name_hash },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Deleted ${secret.name}`)
					onDeleted()
				},
				onError: (err) => toast.error(err.message || "Failed to delete secret"),
			},
		)
	}

	return (
		<div className="flex-1 overflow-y-auto space-y-4 px-6 py-4">
			<div>
				<label className="text-xs font-medium text-muted-foreground">Name</label>
				<div className="mt-1 flex items-center gap-2">
					<code className="flex-1 rounded bg-muted px-2 py-1 font-mono text-xs font-semibold">{secret.name}</code>
					<Button variant="ghost" size="icon-sm" onClick={() => handleCopy(secret.name, "name")}>
						{copied === "name" ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
					</Button>
				</div>
			</div>

			<div>
				<label className="text-xs font-medium text-muted-foreground">Value</label>
				<div className="mt-1 flex items-start gap-2">
					<code className="flex-1 break-all rounded bg-muted px-2 py-1 font-mono text-xs">{secret.value}</code>
					<Button variant="ghost" size="icon-sm" onClick={() => handleCopy(secret.value, "value")}>
						{copied === "value" ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
					</Button>
				</div>
			</div>

			<div className="flex gap-6">
				<div>
					<label className="text-xs font-medium text-muted-foreground">Version</label>
					<p className="mt-1 text-sm">v{secret.version ?? 1}</p>
				</div>
				<div>
					<label className="text-xs font-medium text-muted-foreground">Last Updated</label>
					<p className="mt-1 text-sm">
						{secret.updated_at ? new Date(secret.updated_at).toLocaleString() : "—"}
					</p>
				</div>
			</div>

			<div>
				<label className="text-xs font-medium text-muted-foreground">Copyable</label>
				<div className="mt-1 flex items-center gap-2">
					<code className="flex-1 truncate rounded bg-muted px-2 py-1 font-mono text-xs">
						{secret.name}={secret.value}
					</code>
					<Button variant="ghost" size="icon-sm" onClick={() => handleCopy(`${secret.name}=${secret.value}`, "env")}>
						{copied === "env" ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
					</Button>
				</div>
			</div>

			<div className="flex gap-2 pt-2">
				<Button variant="outline" size="sm" onClick={onEdit}>
					<Pencil /> Edit value
				</Button>
			</div>

			<Separator />

			{/* Version History */}
			<VersionHistory
				projectId={projectId}
				environment={environment}
				nameHash={secret.name_hash}
			/>

			<Separator />

			{/* Danger Zone — type-to-confirm delete */}
			<div className="rounded-lg border border-destructive/30 p-4">
				<div className="flex items-center justify-between">
					<div>
						<p className="text-sm font-medium">Delete this secret</p>
						<p className="mt-0.5 text-xs text-muted-foreground">
							This action cannot be undone.
						</p>
					</div>
					<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
						<DialogTrigger render={<Button variant="danger" size="sm">Delete</Button>} />
						<DialogContent>
							<DialogHeader>
								<DialogTitle>Delete {secret.name}?</DialogTitle>
								<DialogDescription>
									This will permanently delete the secret and all its version history.
								</DialogDescription>
							</DialogHeader>

							<div className="py-2">
								<label className="text-xs font-medium text-muted-foreground">
									Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{secret.name}</code> to confirm
								</label>
								<Input
									className="mt-1.5"
									placeholder={secret.name}
									value={confirmText}
									onChange={(e) => setConfirmText(e.target.value)}
									autoFocus
								/>
							</div>

							<DialogFooter>
								<DialogClose>
									<Button variant="ghost" size="sm" type="button">Cancel</Button>
								</DialogClose>
								<Button
									variant="danger"
									size="sm"
									disabled={confirmText !== secret.name}
									isLoading={deleteSecret.isPending}
									onClick={handleDelete}
								>
									Delete permanently
								</Button>
							</DialogFooter>

							{deleteSecret.error && (
								<Alert variant="danger" className="mt-2">
									<AlertCircle />
									<AlertDescription>{deleteSecret.error.message}</AlertDescription>
								</Alert>
							)}
						</DialogContent>
					</Dialog>
				</div>
			</div>
		</div>
	)
}

function VersionHistory({ projectId, environment, nameHash }: {
	projectId: string
	environment: string
	nameHash: string
}) {
	const { data, isLoading } = useSecretVersions(projectId, environment, nameHash)
	const rollback = useRollbackSecret()

	const versions = data?.versions ?? []
	const currentVersion = data?.current_version

	return (
		<div>
			<div className="flex items-center gap-2">
				<History className="size-3.5 text-muted-foreground" />
				<label className="text-xs font-medium text-muted-foreground">Version History</label>
			</div>

			{isLoading ? (
				<div className="flex items-center justify-center py-4"><Spinner /></div>
			) : versions.length === 0 ? (
				<p className="mt-2 text-xs text-muted-foreground">No previous versions.</p>
			) : (
				<div className="mt-2 space-y-1.5">
					{/* Current version */}
					<div className="flex items-center justify-between rounded-md bg-muted/50 px-2.5 py-1.5">
						<div className="flex items-center gap-2">
							<Badge variant="neutral" className="text-[10px]">v{currentVersion}</Badge>
							<span className="text-xs text-muted-foreground">current</span>
						</div>
					</div>

					{/* Archived versions */}
					{versions.map((v) => (
						<div key={v.version} className="flex items-center justify-between rounded-md bg-muted/30 px-2.5 py-1.5">
							<div className="flex items-center gap-2">
								<Badge variant="neutral" className="text-[10px]">v{v.version}</Badge>
								<span className="text-xs text-muted-foreground">
									{v.created_at ? new Date(v.created_at).toLocaleString() : ""}
								</span>
							</div>
							<AlertDialog>
								<AlertDialogTrigger
									render={
										<Button
											variant="ghost"
											size="icon-sm"
											className="size-6 text-muted-foreground hover:text-foreground"
											title={`Rollback to v${v.version}`}
										/>
									}
								>
									<RotateCcw className="size-3" />
								</AlertDialogTrigger>
								<AlertDialogContent>
									<AlertDialogHeader>
										<AlertDialogTitle>Rollback to v{v.version}</AlertDialogTitle>
										<AlertDialogDescription>
											This will revert the secret to version {v.version}. The current version will be archived.
										</AlertDialogDescription>
									</AlertDialogHeader>
									<AlertDialogFooter>
										<AlertDialogCancel>Cancel</AlertDialogCancel>
										<AlertDialogAction
											onClick={() => rollback.mutate(
												{ projectId, environment, nameHash, version: v.version! },
												{
													onSuccess: () => toast.success(`Rolled back to v${v.version}`),
													onError: (err) => toast.error(err.message || "Rollback failed"),
												},
											)}
										>
											{rollback.isPending ? <Spinner className="animate-spin" /> : "Rollback"}
										</AlertDialogAction>
									</AlertDialogFooter>
								</AlertDialogContent>
							</AlertDialog>
						</div>
					))}
				</div>
			)}
		</div>
	)
}

function DeleteSecretButton({ projectId, environment, secret }: {
	projectId: string
	environment: string
	secret: DecryptedSecret
}) {
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const deleteSecret = useDeleteSecret()

	const handleDelete = () => {
		deleteSecret.mutate(
			{ projectId, environment, nameHash: secret.name_hash },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Deleted ${secret.name}`)
				},
				onError: (err) => toast.error(err.message || "Failed to delete secret"),
			},
		)
	}

	return (
		<Dialog open={confirmOpen} onOpenChange={(v) => { setConfirmOpen(v); if (!v) { setConfirmText(""); deleteSecret.reset() } }}>
			<DialogTrigger
				render={
					<Button
						variant="ghost"
						size="icon-sm"
						className="text-muted-foreground hover:text-destructive"
						type="button"
						onClick={(e) => e.stopPropagation()}
					/>
				}
			>
				<Trash2 />
			</DialogTrigger>
			<DialogContent onClick={(e) => e.stopPropagation()}>
				<DialogHeader>
					<DialogTitle>Delete {secret.name}?</DialogTitle>
					<DialogDescription>
						This will permanently delete the secret and all its version history.
					</DialogDescription>
				</DialogHeader>

				<div className="py-2">
					<label className="text-xs font-medium text-muted-foreground">
						Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{secret.name}</code> to confirm
					</label>
					<Input
						className="mt-1.5"
						placeholder={secret.name}
						value={confirmText}
						onChange={(e) => setConfirmText(e.target.value)}
						autoFocus
					/>
				</div>

				<DialogFooter>
					<DialogClose>
						<Button variant="ghost" size="sm" type="button">Cancel</Button>
					</DialogClose>
					<Button
						variant="danger"
						size="sm"
						disabled={confirmText !== secret.name}
						isLoading={deleteSecret.isPending}
						onClick={handleDelete}
					>
						Delete permanently
					</Button>
				</DialogFooter>

				{deleteSecret.error && (
					<Alert variant="danger" className="mt-2">
						<AlertCircle />
						<AlertDescription>{deleteSecret.error.message}</AlertDescription>
					</Alert>
				)}
			</DialogContent>
		</Dialog>
	)
}

function MaskedValue({ value }: { value: string }) {
	const [revealed, setRevealed] = useState(false)
	return (
		<div className="flex items-center gap-1.5">
			{revealed ? (
				<code className="max-w-50 truncate font-mono text-xs">{value}</code>
			) : (
				<span className="text-xs text-muted-foreground">{"•".repeat(Math.min(value.length, 20))}</span>
			)}
			<Button
				variant="ghost"
				size="icon-sm"
				className="size-5"
				onClick={(e) => { e.stopPropagation(); setRevealed((r) => !r) }}
			>
				{revealed ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
			</Button>
		</div>
	)
}

function PageHeader() {
	return (
		<div>
			<h1 className="text-lg font-semibold">Secrets</h1>
			<p className="mt-1 text-sm text-muted-foreground">Encrypted on your device. Decrypted in your browser.</p>
		</div>
	)
}
