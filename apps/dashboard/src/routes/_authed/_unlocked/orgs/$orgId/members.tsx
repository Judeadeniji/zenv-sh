import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { z } from "zod"
import type { ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Avatar } from "#/components/ui/avatar"
import { Spinner } from "#/components/ui/spinner"
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
} from "#/components/ui/sheet"
import { DataTable } from "#/components/data-table"
import { SearchInput } from "#/components/search-input"
import { InviteMemberDialog } from "#/components/invite-member-dialog"
import { orgMembersQueryOptions, useRemoveMember } from "#/lib/queries/orgs"
import { meQueryOptions } from "#/lib/queries/auth"
import { Users, UserPlus, Trash2 } from "lucide-react"
import { getInitials } from "#/lib/utils"
import { formatDateTime } from "#/lib/format"

const searchSchema = z
	.object({
		limit: z.number().default(50),
		offset: z.number().default(0),
		sortBy: z.string().default("createdAt"),
		sortDir: z.enum(["asc", "desc"]).default("desc"),
		filterField: z.string().default(""),
		filterValue: z.string().default(""),
		filterOperator: z
			.enum([
				"eq",
				"ne",
				"gt",
				"gte",
				"lt",
				"lte",
				"in",
				"not_in",
				"contains",
				"starts_with",
				"ends_with",
			])
			.default("eq"),
	})
	.partial()

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/members")({
	validateSearch: searchSchema,
	component: MembersPage,
})

function MembersPage() {
	const { orgId } = Route.useParams()
	const search = Route.useSearch()
	const navigate = useNavigate({ from: Route.fullPath })

	const { data: me } = useQuery(meQueryOptions)
	const { data, isLoading } = useQuery(orgMembersQueryOptions(orgId, search))
	const removeMember = useRemoveMember()
	const [selectedMember, setSelectedMember] = useState<MemberRow | null>(null)

	const members = data?.members ?? []

	type MemberRow = (typeof members)[number]

	const columns: ColumnDef<MemberRow, unknown>[] = [
		{
			accessorKey: "name",
			header: "Member",
			cell: ({ row }) => {
				const m = row.original.user
				const isMe = m.email === me?.email
				return (
					<div className="flex items-center gap-3">
						<Avatar size="sm" fallback={getInitials(m.name, m.email)} />
						<div>
							<p className="text-sm font-medium">
								{m.name}
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
					{formatDateTime(row.original.createdAt.toISOString())}
				</span>
			),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) => {
				const isMe = row.original.user.email === me?.email
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
				<div className="flex items-center justify-center py-20">
					<Spinner />
				</div>
			</div>
		)
	}

	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<PageHeader />
				<InviteMemberDialog
					orgId={orgId}
					trigger={
						<Button type="button" size="sm">
							<UserPlus /> Invite
						</Button>
					}
				/>
			</div>

			<div className="mb-4 flex items-center gap-3">
				<SearchInput
					placeholder="Search members..."
					value={search.filterValue}
					onChange={(val) => {
						navigate({
							search: (prev) => ({ ...prev, search: val || undefined, page: 1 }),
							replace: true,
						})
					}}
				/>
			</div>

			<DataTable
				columns={columns}
				data={members}
				onRowClick={(row) => setSelectedMember(row.original)}
				emptyIcon={<Users />}
				emptyTitle="Just you for now"
				emptyDescription="Invite team members to collaborate. Everyone sets up their own vault — no one can see anyone else's Vault Key."
				emptyAction={
					<InviteMemberDialog
						orgId={orgId}
						trigger={
							<Button type="button" size="sm">
								<UserPlus /> Invite a member
							</Button>
						}
					/>
				}
			/>

			<Sheet
				open={!!selectedMember}
				onOpenChange={(open) => {
					if (!open) setSelectedMember(null)
				}}
			>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>{selectedMember?.user.name || "Unnamed"}</SheetTitle>
						<SheetDescription>{selectedMember?.user.email}</SheetDescription>
					</SheetHeader>
					{selectedMember && (
						<div className="space-y-4 px-6 py-4">
							<div>
								<span className="text-xs font-medium text-muted-foreground">Role</span>
								<p className="mt-1">
									<Badge variant={selectedMember.role === "admin" ? "primary" : "neutral"}>
										{selectedMember.role}
									</Badge>
								</p>
							</div>
							<div>
								<span className="text-xs font-medium text-muted-foreground">Joined</span>
								<p className="mt-1 text-sm">
									{formatDateTime(selectedMember.createdAt.toISOString())}
								</p>
							</div>
							{selectedMember.user.email !== me?.email && selectedMember.role !== "owner" && (
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
