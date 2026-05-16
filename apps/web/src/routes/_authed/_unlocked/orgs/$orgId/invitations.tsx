import { useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
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
import { InviteMemberDialog } from "#/components/invite-member-dialog"
import { orgInvitationQueries, useRemoveMember } from "#/lib/queries/orgs"
import { Mail, UserPlus, Trash2, Clock } from "lucide-react"
import { getInitials } from "#/lib/utils"
import type { InvitationStatus } from "better-auth/plugins"

const searchSchema = z
	.object({
		page: z.number().default(1),
		per_page: z.number().default(50),
		search: z.string().default(""),
		role: z.string().default(""),
		sort_by: z.string().default("createdAt"),
		sort_dir: z.enum(["asc", "desc"]).default("desc"),
	})
	.partial()

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/invitations")({
	validateSearch: searchSchema,
	component: InvitationsPage,
})

interface InvitationRow {
	id: string
	organizationId: string
	email: string
	role: "admin" | "member" | "owner"
	status: InvitationStatus
	inviterId: string
	expiresAt: Date | string
	createdAt: Date | string
}

function InvitationsPage() {
	const { orgId } = Route.useParams()
	const { data: invitations, isLoading } = useQuery(orgInvitationQueries(orgId))
	const [selectedInvitation, setSelectedInvitation] = useState<InvitationRow | null>(null)

	const columns: ColumnDef<InvitationRow, unknown>[] = [
		{
			accessorKey: "email",
			header: "Invitee",
			cell: ({ row }) => {
				const inv = row.original
				return (
					<div className="flex items-center gap-3">
						<Avatar size="sm" fallback={getInitials(undefined, inv.email)} />
						<div>
							<p className="text-sm font-medium">{inv.email}</p>
							<p className="text-xs text-muted-foreground flex items-center gap-1">
								<Clock className="size-3" />
								Invited {new Date(inv.createdAt).toLocaleDateString()}
							</p>
						</div>
					</div>
				)
			},
		},
		{
			accessorKey: "status",
			header: "Status",
			cell: ({ row }) => {
				const status = row.original.status
				const variants: Record<string, "neutral" | "primary" | "danger" | "warning"> = {
					pending: "warning",
					accepted: "primary",
					rejected: "danger",
					expired: "neutral",
				}
				return <Badge variant={variants[status] ?? "neutral"}>{status}</Badge>
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
			id: "actions",
			header: "",
			cell: ({ row }) => <RowActions orgId={orgId} invitation={row.original} />,
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

			<DataTable
				columns={columns}
				data={invitations ?? []}
				onRowClick={(row) => setSelectedInvitation(row.original)}
				emptyIcon={<Mail />}
				emptyTitle="No pending invitations"
				emptyDescription="When you invite team members, their pending status will appear here until they accept."
				emptyAction={
					<InviteMemberDialog
						orgId={orgId}
						trigger={
							<Button type="button" size="sm">
								<UserPlus /> Send an invitation
							</Button>
						}
					/>
				}
			/>

			<InvitationDetailsSheet
				orgId={orgId}
				invitation={selectedInvitation}
				onClose={() => setSelectedInvitation(null)}
			/>
		</div>
	)
}

/**
 * Extracted Actions component to handle mutation logic cleanly within the table
 */
function RowActions({ orgId, invitation }: { orgId: string; invitation: InvitationRow }) {
	const removeMember = useRemoveMember()
	return (
		<div className="text-right">
			<Button
				variant="ghost"
				size="icon-sm"
				className="text-muted-foreground hover:text-destructive"
				onClick={(e) => {
					e.stopPropagation()
					removeMember.mutate({ orgId, memberId: invitation.id })
				}}
			>
				<Trash2 className="size-3.5" />
			</Button>
		</div>
	)
}

/**
 * Extracted Sheet Component
 */
interface InvitationDetailsSheetProps {
	orgId: string
	invitation: InvitationRow | null
	onClose: () => void
}

function InvitationDetailsSheet({ orgId, invitation, onClose }: InvitationDetailsSheetProps) {
	const removeMember = useRemoveMember()

	return (
		<Sheet
			open={!!invitation}
			onOpenChange={(open) => {
				if (!open) onClose()
			}}
		>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Invitation Details</SheetTitle>
					<SheetDescription>{invitation?.email}</SheetDescription>
				</SheetHeader>
				{invitation && (
					<div className="space-y-4 px-6 py-4">
						<div>
							<span className="text-xs font-medium text-muted-foreground">Status</span>
							<p className="mt-1">
								<Badge variant={invitation.status === "pending" ? "warning" : "neutral"}>
									{invitation.status}
								</Badge>
							</p>
						</div>
						<div>
							<span className="text-xs font-medium text-muted-foreground">Role</span>
							<p className="mt-1">
								<Badge variant={invitation.role === "admin" ? "primary" : "neutral"}>
									{invitation.role}
								</Badge>
							</p>
						</div>
						<div>
							<span className="text-xs font-medium text-muted-foreground">Sent On</span>
							<p className="mt-1 text-sm">{new Date(invitation.createdAt).toLocaleString()}</p>
						</div>
						<div>
							<span className="text-xs font-medium text-muted-foreground">Expires</span>
							<p className="mt-1 text-sm text-muted-foreground">
								{new Date(invitation.expiresAt).toLocaleString()}
							</p>
						</div>
						<div className="pt-2">
							<Button
								variant="danger"
								size="sm"
								onClick={() => {
									removeMember.mutate(
										{ orgId, memberId: invitation.id },
										{
											onSuccess: () => onClose(),
										},
									)
								}}
								isLoading={removeMember.isPending}
							>
								<Trash2 /> Revoke Invitation
							</Button>
						</div>
					</div>
				)}
			</SheetContent>
		</Sheet>
	)
}

function PageHeader() {
	return (
		<div>
			<h1 className="text-lg font-semibold">Invitations</h1>
			<p className="mt-1 text-sm text-muted-foreground">
				Manage pending requests for people to join your organization.
			</p>
		</div>
	)
}
