import { useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { DataTable } from "#/components/data-table"
import { CreateSecretDialog } from "#/components/create-secret-dialog"
import { secretsQueryOptions, useDeleteSecret } from "#/lib/queries/secrets"
import { useNavStore } from "#/lib/stores/nav"
import { KeyRound, Plus, Upload, EyeOff, Trash2 } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/secrets")({
	component: SecretsPage,
})

interface SecretRow {
	name_hash: string
	encrypted_name?: string
	created_at?: string
	updated_at?: string
}

function SecretsPage() {
	const { projectId } = Route.useParams()
	const environment = useNavStore((s) => s.activeEnvironment)
	const { data, isLoading } = useQuery(secretsQueryOptions(projectId, environment))
	const deleteSecret = useDeleteSecret()
	const [selectedSecret, setSelectedSecret] = useState<SecretRow | null>(null)

	const secrets: SecretRow[] = (data as { secrets?: SecretRow[] })?.secrets ?? []

	const columns: ColumnDef<SecretRow, unknown>[] = [
		{
			accessorKey: "name_hash",
			header: "Name",
			cell: ({ row }) => (
				<code className="rounded bg-muted px-1.5 py-0.5 text-xs font-medium">
					{row.original.name_hash?.slice(0, 16)}...
				</code>
			),
		},
		{
			id: "value",
			header: "Value",
			cell: () => (
				<Badge variant="neutral" className="font-mono text-xs">
					<EyeOff className="size-3" /> encrypted
				</Badge>
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
							deleteSecret.mutate({ projectId, nameHash: row.original.name_hash })
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
				<PageHeader />
				<div className="flex items-center justify-center py-20"><Spinner /></div>
			</div>
		)
	}

	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<PageHeader />
				<CreateSecretDialog
					projectId={projectId}
					trigger={<Button type="button" size="sm"><Plus /> Add</Button>}
				/>
			</div>

			<DataTable
				columns={columns}
				data={secrets}
				filterColumn="name_hash"
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
						<Button variant="outline" size="sm"><Upload /> Import</Button>
					</div>
				}
			/>

			<Sheet open={!!selectedSecret} onOpenChange={(open) => { if (!open) setSelectedSecret(null) }}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Secret Details</SheetTitle>
						<SheetDescription>Encrypted secret — decrypt with your Vault Key to view.</SheetDescription>
					</SheetHeader>
					{selectedSecret && (
						<div className="space-y-4 px-6 py-4">
							<div>
								<label className="text-xs font-medium text-muted-foreground">Name Hash</label>
								<p className="mt-1 font-mono text-xs break-all">{selectedSecret.name_hash}</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Created</label>
								<p className="mt-1 text-sm">
									{selectedSecret.created_at ? new Date(selectedSecret.created_at).toLocaleString() : "—"}
								</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Last Updated</label>
								<p className="mt-1 text-sm">
									{selectedSecret.updated_at ? new Date(selectedSecret.updated_at).toLocaleString() : "—"}
								</p>
							</div>
						</div>
					)}
				</SheetContent>
			</Sheet>
		</div>
	)
}

function PageHeader() {
	return (
		<div>
			<h1 className="text-lg font-semibold">Secrets</h1>
			<p className="mt-1 text-sm text-muted-foreground">Encrypted secrets for your project.</p>
		</div>
	)
}
