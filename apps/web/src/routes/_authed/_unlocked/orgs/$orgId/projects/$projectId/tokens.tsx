import { useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { DataTable } from "#/components/data-table"
import { CreateTokenDialog } from "#/components/create-token-dialog"
import { tokensQueryOptions, useRevokeToken } from "#/lib/queries/tokens"
import { FileKey, Plus, Trash2 } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/tokens")({
	component: TokensPage,
})

interface TokenRow {
	id: string
	name: string
	permission?: string
	environment?: string
	created_at?: string
	last_used_at?: string
}

function TokensPage() {
	const { projectId } = Route.useParams()
	const { data, isLoading } = useQuery(tokensQueryOptions(projectId))
	const revoke = useRevokeToken()
	const [selectedToken, setSelectedToken] = useState<TokenRow | null>(null)

	const tokens: TokenRow[] = (data as { tokens?: TokenRow[] })?.tokens ?? []

	const columns: ColumnDef<TokenRow, unknown>[] = [
		{
			accessorKey: "name",
			header: "Name",
			cell: ({ row }) => <span className="text-sm font-medium">{row.original.name}</span>,
		},
		{
			accessorKey: "permission",
			header: "Permission",
			cell: ({ row }) => (
				<Badge variant={row.original.permission === "read_write" ? "warning" : "neutral"}>
					{row.original.permission === "read_write" ? "Read & Write" : "Read"}
				</Badge>
			),
		},
		{
			accessorKey: "environment",
			header: "Environment",
			cell: ({ row }) => <Badge variant="neutral">{row.original.environment}</Badge>,
		},
		{
			accessorKey: "last_used_at",
			header: "Last Used",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{row.original.last_used_at ? new Date(row.original.last_used_at).toLocaleDateString() : "Never"}
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
							revoke.mutate({ projectId, tokenId: row.original.id })
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
				<CreateTokenDialog
					projectId={projectId}
					trigger={<Button type="button" size="sm"><Plus /> Create</Button>}
				/>
			</div>

			<DataTable
				columns={columns}
				data={tokens}
				filterColumn="name"
				filterPlaceholder="Search tokens..."
				onRowClick={(row) => setSelectedToken(row.original)}
				emptyIcon={<FileKey />}
				emptyTitle="No service tokens"
				emptyDescription="Service tokens let your applications read secrets without a human in the loop."
				emptyAction={
					<CreateTokenDialog
						projectId={projectId}
						trigger={<Button type="button" size="sm"><Plus /> Create a token</Button>}
					/>
				}
			/>

			<Sheet open={!!selectedToken} onOpenChange={(open) => { if (!open) setSelectedToken(null) }}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>{selectedToken?.name}</SheetTitle>
						<SheetDescription>Service token details</SheetDescription>
					</SheetHeader>
					{selectedToken && (
						<div className="space-y-4 px-6 py-4">
							<div>
								<label className="text-xs font-medium text-muted-foreground">Permission</label>
								<p className="mt-1">
									<Badge variant={selectedToken.permission === "read_write" ? "warning" : "neutral"}>
										{selectedToken.permission === "read_write" ? "Read & Write" : "Read"}
									</Badge>
								</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Environment</label>
								<p className="mt-1 text-sm">{selectedToken.environment}</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Created</label>
								<p className="mt-1 text-sm">
									{selectedToken.created_at ? new Date(selectedToken.created_at).toLocaleString() : "—"}
								</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Last Used</label>
								<p className="mt-1 text-sm">
									{selectedToken.last_used_at ? new Date(selectedToken.last_used_at).toLocaleString() : "Never"}
								</p>
							</div>
							<div className="pt-2">
								<Button
									variant="danger"
									size="sm"
									onClick={() => {
										revoke.mutate({ projectId, tokenId: selectedToken.id })
										setSelectedToken(null)
									}}
									isLoading={revoke.isPending}
								>
									<Trash2 /> Revoke token
								</Button>
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
			<h1 className="text-lg font-semibold">Service Tokens</h1>
			<p className="mt-1 text-sm text-muted-foreground">Programmatic access for your CI/CD pipelines and applications.</p>
		</div>
	)
}
