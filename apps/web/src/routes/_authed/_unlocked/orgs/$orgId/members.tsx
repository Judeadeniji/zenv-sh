import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { z } from "zod"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Avatar } from "#/components/ui/avatar"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { DataTable } from "#/components/data-table"
import { InviteMemberDialog } from "#/components/invite-member-dialog"
import { orgMembersQueryOptions, useRemoveMember } from "#/lib/queries/orgs"
import { meQueryOptions } from "#/lib/queries/auth"
import { Users, UserPlus, Trash2 } from "lucide-react"

const searchSchema = z.object({
	page: z.number().catch(1),
	per_page: z.number().catch(50),
	search: z.string().optional(),
	role: z.string().optional(),
	sort_by: z.string().optional(),
	sort_dir: z.enum(["asc", "desc"]).optional(),
})

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/members")({
	validateSearch: searchSchema,
	component: MembersPage,
})

interface MemberRow {
	id: string
	user_id?: string
	email?: string
	name?: string
	role?: string
	created_at?: string
}

function getInitials(name?: string, email?: string): string {
	const source = name || email || "?"
	return source
		.split(/[\s@]/)
		.slice(0, 2)
		.map((s) => s[0]?.toUpperCase() ?? "")
		.join("")
}

function MembersPage() {
	const { orgId } = Route.useParams()
	const search = Route.useSearch()
	const navigate = useNavigate({ from: Route.fullPath })

	const { data: me } = useQuery(meQueryOptions)
	const { data, isLoading } = useQuery(orgMembersQueryOptions(orgId, search))
	const removeMember = useRemoveMember()
	const [selectedMember, setSelectedMember] = useState<MemberRow | null>(null)

	const members: MemberRow[] = (data as { members?: MemberRow[] })?.members ?? []

	const columns: ColumnDef<MemberRow, unknown>[] = [
		{
			accessorKey: "name",
			header: "Member",
			cell: ({ row }) => {
				const m = row.original
				const isMe = m.email === me?.email
				return (
					<div className="flex items-center gap-3">
						<Avatar size="sm" fallback={getInitials(m.name, m.email)} />
						<div>
							<p className="text-sm font-medium">
								{m.name || "Unnamed"}
								{isMe && <span className="ml-1 text-xs text-muted-foreground">(you)</span>}
							</p>
							<p className="text-xs text-muted-foreground">{m.email}</p>
						</div>
					</div>
				)
			},
		},
		{
			accessorKey: "role",
			header: "Role",
			cell: ({ row }) => (
				<Badge variant={row.original.role === "admin" ? "primary" : "neutral"}>
					{row.original.role}
				</Badge>
			),
		},
		{
			accessorKey: "created_at",
			header: "Joined",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{row.original.created_at ? new Date(row.original.created_at).toLocaleDateString() : "—"}
				</span>
			),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) => {
				const isMe = row.original.email === me?.email
				if (isMe) return null
				return (
					<div className="text-right">
						<Button
							variant="ghost"
							size="icon-sm"
							className="text-muted-foreground hover:text-destructive"
							onClick={(e) => {
								e.stopPropagation()
								removeMember.mutate({ orgId, memberId: row.original.id })
							}}
						>
							<Trash2 className="size-3.5" />
						</Button>
					</div>
				)
			},
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
				<InviteMemberDialog
					orgId={orgId}
					trigger={<Button type="button" size="sm"><UserPlus /> Invite</Button>}
				/>
			</div>

			<DataTable
				columns={columns}
				data={members}
				filterColumn="name"
				searchValue={search.search}
				onSearchChange={(val) => {
					navigate({ search: (prev) => ({ ...prev, search: val || undefined, page: 1 }), replace: true })
				}}
				pagination={data?.meta ? {
					page: data.meta.page ?? 1,
					totalPages: data.meta.total_pages ?? 1,
					total: data.meta.total ?? 0,
					onPageChange: (p) => navigate({ search: (prev) => ({ ...prev, page: p }) })
				} : undefined}
				filterPlaceholder="Search members..."
				onRowClick={(row) => setSelectedMember(row.original)}
				emptyIcon={<Users />}
				emptyTitle="Just you for now"
				emptyDescription="Invite team members to collaborate. Everyone sets up their own vault — no one can see anyone else's Vault Key."
				emptyAction={
					<InviteMemberDialog
						orgId={orgId}
						trigger={<Button type="button" size="sm"><UserPlus /> Invite a member</Button>}
					/>
				}
			/>

			<Sheet open={!!selectedMember} onOpenChange={(open) => { if (!open) setSelectedMember(null) }}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>{selectedMember?.name || "Unnamed"}</SheetTitle>
						<SheetDescription>{selectedMember?.email}</SheetDescription>
					</SheetHeader>
					{selectedMember && (
						<div className="space-y-4 px-6 py-4">
							<div>
								<label className="text-xs font-medium text-muted-foreground">Role</label>
								<p className="mt-1">
									<Badge variant={selectedMember.role === "admin" ? "primary" : "neutral"}>
										{selectedMember.role}
									</Badge>
								</p>
							</div>
							<div>
								<label className="text-xs font-medium text-muted-foreground">Joined</label>
								<p className="mt-1 text-sm">
									{selectedMember.created_at ? new Date(selectedMember.created_at).toLocaleString() : "—"}
								</p>
							</div>
							{selectedMember.email !== me?.email && (
								<div className="pt-2">
									<Button
										variant="danger"
										size="sm"
										onClick={() => {
											removeMember.mutate({ orgId, memberId: selectedMember.id })
											setSelectedMember(null)
										}}
										isLoading={removeMember.isPending}
									>
										<Trash2 /> Remove member
									</Button>
								</div>
							)}
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
			<h1 className="text-lg font-semibold">Members</h1>
			<p className="mt-1 text-sm text-muted-foreground">People in your organization.</p>
		</div>
	)
}
