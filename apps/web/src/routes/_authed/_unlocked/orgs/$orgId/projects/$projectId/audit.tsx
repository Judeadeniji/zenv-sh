import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { DataTable } from "#/components/data-table"
import { auditQueryOptions } from "#/lib/queries/audit"
import { Shield } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/audit")({
	component: AuditPage,
})

interface AuditRow {
	id: string
	action?: string
	actor_email?: string
	resource?: string
	result?: string
	created_at?: string
}

function AuditPage() {
	const { projectId } = Route.useParams()
	const { data, isLoading } = useQuery(auditQueryOptions(projectId))

	const logs: AuditRow[] = (data as { logs?: AuditRow[] })?.logs ?? []

	const columns: ColumnDef<AuditRow, unknown>[] = [
		{
			accessorKey: "action",
			header: "Action",
			cell: ({ row }) => <Badge variant="neutral">{row.original.action}</Badge>,
		},
		{
			accessorKey: "actor_email",
			header: "Actor",
			cell: ({ row }) => <span className="text-xs text-muted-foreground">{row.original.actor_email}</span>,
		},
		{
			accessorKey: "resource",
			header: "Resource",
			cell: ({ row }) => (
				<code className="rounded bg-muted px-1.5 py-0.5 text-xs">{row.original.resource}</code>
			),
		},
		{
			accessorKey: "result",
			header: "Result",
			cell: ({ row }) => (
				<Badge variant={row.original.result === "success" ? "success" : row.original.result === "denied" ? "danger" : "neutral"}>
					{row.original.result}
				</Badge>
			),
		},
		{
			accessorKey: "created_at",
			header: "Time",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{row.original.created_at ? new Date(row.original.created_at).toLocaleString() : "—"}
				</span>
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
			<div className="mb-6">
				<PageHeader />
			</div>

			<DataTable
				columns={columns}
				data={logs}
				filterColumn="action"
				filterPlaceholder="Filter by action..."
				emptyIcon={<Shield />}
				emptyTitle="No activity yet"
				emptyDescription="Every secret access, change, and token usage is logged here automatically. Activity will appear once you start using your project."
			/>
		</div>
	)
}

function PageHeader() {
	return (
		<div>
			<h1 className="text-lg font-semibold">Audit Log</h1>
			<p className="mt-1 text-sm text-muted-foreground">A record of every action in your project.</p>
		</div>
	)
}
