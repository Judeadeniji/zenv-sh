import { useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { DataTable } from "#/components/data-table"
import { CreateSecretDialog } from "#/components/create-secret-dialog"
import { ImportSecretsDialog } from "#/components/import-secrets-dialog"
import { useDecryptedSecrets, useDeleteSecret } from "#/lib/queries/secrets"
import { useNavStore } from "#/lib/stores/nav"
import { KeyRound, Plus, Upload, Eye, EyeOff, Trash2, Copy, Check } from "lucide-react"

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
	const deleteSecret = useDeleteSecret()
	const [selectedSecret, setSelectedSecret] = useState<DecryptedSecret | null>(null)

	const rows = secrets ?? []

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
				<div className="text-right">
					<Button
						variant="ghost"
						size="icon-sm"
						className="text-muted-foreground hover:text-destructive"
						onClick={(e) => {
							e.stopPropagation()
							deleteSecret.mutate({ projectId, environment, nameHash: row.original.name_hash })
						}}
					>
						<Trash2 className="size-3.5" />
					</Button>
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

			<DataTable
				columns={columns}
				data={rows}
				filterColumn="name"
				filterPlaceholder="Search secrets..."
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
					{selectedSecret && <SecretDetailSheet secret={selectedSecret} />}
				</SheetContent>
			</Sheet>
		</div>
	)
}

function SecretDetailSheet({ secret }: { secret: DecryptedSecret }) {
	const [copied, setCopied] = useState<string | null>(null)

	const handleCopy = (text: string, key: string) => {
		navigator.clipboard?.writeText(text).then(() => {
			setCopied(key)
			setTimeout(() => setCopied(null), 2000)
		})
	}

	return (
		<div className="space-y-4 px-6 py-4">
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
				<label className="text-xs font-medium text-muted-foreground">.env format</label>
				<div className="mt-1 flex items-center gap-2">
					<code className="flex-1 truncate rounded bg-muted px-2 py-1 font-mono text-xs">
						{secret.name}={secret.value}
					</code>
					<Button variant="ghost" size="icon-sm" onClick={() => handleCopy(`${secret.name}=${secret.value}`, "env")}>
						{copied === "env" ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
					</Button>
				</div>
			</div>
		</div>
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
