import { useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { DataTable } from "#/components/data-table"
import { auditQueryOptions } from "#/lib/queries/audit"
import { formatRelativeTime } from "#/lib/format"
import { env } from "#/lib/env"
import { Shield, Download } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/audit")({
	component: AuditPage,
})

interface AuditRow {
	id: string
	action?: string
	actor_email?: string
	user_id?: string
	token_id?: string
	ip?: string
	result?: string
	created_at?: string
}

function AuditPage() {
	const { projectId } = Route.useParams()
	const [page, setPage] = useState(1)
	const perPage = 50

	const { data, isLoading } = useQuery(auditQueryOptions(projectId, { page, perPage }))
	const resp = data as { entries?: AuditRow[]; total?: number; page?: number; per_page?: number } | undefined
	const entries = resp?.entries ?? []
	const total = resp?.total ?? 0
	const totalPages = Math.max(1, Math.ceil(total / perPage))

	const handleExport = () => {
		const url = `${env.VITE_API_URL}/audit-logs/export?project_id=${projectId}`
		window.open(url, "_blank")
	}

	const columns: ColumnDef<AuditRow, unknown>[] = [
		{
			accessorKey: "action",
			header: "Action",
			cell: ({ row }) => <Badge variant="neutral" className="font-mono text-[11px]">{row.original.action}</Badge>,
		},
		{
			id: "actor",
			header: "Actor",
			cell: ({ row }) => {
				const { actor_email, user_id, token_id } = row.original
				return (
					<div className="flex items-center gap-1.5">
						<span className="truncate text-xs text-muted-foreground">
							{actor_email ?? user_id?.slice(0, 8) ?? "—"}
						</span>
						{token_id && <Badge variant="neutral" className="text-[10px]">token</Badge>}
					</div>
				)
			},
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
			accessorKey: "ip",
			header: "IP",
			cell: ({ row }) => (
				<span className="font-mono text-xs text-muted-foreground">{row.original.ip ?? "—"}</span>
			),
		},
		{
			accessorKey: "created_at",
			header: "Time",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{row.original.created_at ? formatRelativeTime(row.original.created_at) : "—"}
				</span>
			),
		},
	]

	if (isLoading) {
		return (
			<div>
				<PageHeader onExport={handleExport} />
				<div className="flex items-center justify-center py-20"><Spinner /></div>
			</div>
		)
	}

	return (
		<div>
			<div className="mb-6">
				<PageHeader onExport={handleExport} />
			</div>

			<DataTable
				columns={columns}
				data={entries}
				filterColumn="action"
				filterPlaceholder="Filter by action..."
				pagination={{
					page,
					totalPages,
					total,
					onPageChange: setPage,
				}}
				emptyIcon={<Shield />}
				emptyTitle="No activity yet"
				emptyDescription="Every secret access, change, and token usage is logged here automatically. Activity will appear once you start using your project."
			/>
		</div>
	)
}

function PageHeader({ onExport }: { onExport: () => void }) {
	return (
		<div className="flex items-center justify-between">
			<div>
				<h1 className="text-lg font-semibold">Audit Log</h1>
				<p className="mt-1 text-sm text-muted-foreground">A record of every action in your project.</p>
			</div>
			<Button variant="outline" size="sm" onClick={onExport}>
				<Download /> Export CSV
			</Button>
		</div>
	)
}
