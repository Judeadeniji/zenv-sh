import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Badge } from "#/components/ui/badge"
import {
	Dialog,
	DialogTrigger,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
	DialogFooter,
	DialogClose,
} from "#/components/ui/dialog"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { api } from "#/lib/api-client"
import { useNavigate } from "@tanstack/react-router"
import { orgQueryOptions, orgMembersQueryOptions } from "#/lib/queries/orgs"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { AlertCircle, CheckCircle, Trash2, Crown } from "lucide-react"

// ── Schemas ──

const renameSchema = z.object({
	name: z.string().min(1, "Name is required").max(64, "Name too long"),
})

const deleteSchema = z.object({
	confirmation: z.string(),
})

type RenameInput = z.infer<typeof renameSchema>
type DeleteInput = z.infer<typeof deleteSchema>

// ── Component ──

export function OrgSection({ orgId }: { orgId: string }) {
	return (
		<div>
			<RenameRow orgId={orgId} />
			<SettingsDivider />
			<MembersRow orgId={orgId} />
			<SettingsDivider />
			<DangerRow orgId={orgId} />
		</div>
	)
}

// ── Rename ──

function RenameRow({ orgId }: { orgId: string }) {
	const qc = useQueryClient()
	const { data: org } = useQuery(orgQueryOptions(orgId))
	const orgName = (org as { name?: string })?.name ?? ""

	const form = useForm<RenameInput>({
		resolver: zodResolver(renameSchema),
		values: { name: orgName },
	})

	const rename = useMutation({
		mutationKey: mutationKeys.orgs.rename,
		mutationFn: async (data: RenameInput) => {
			const { error } = await api().PUT("/orgs/{orgID}", {
				params: { path: { orgID: orgId } },
				body: { name: data.name },
			})
			if (error) throw new Error("Failed to rename organization")
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: queryKeys.orgs.all })
			qc.invalidateQueries({ queryKey: queryKeys.orgs.detail(orgId) })
		},
	})

	return (
		<SettingsRow title="Organization name" description="This name is visible to all members.">
			<form onSubmit={form.handleSubmit((d) => rename.mutate(d))} className="space-y-4">
				{rename.isSuccess && (
					<Alert variant="success">
						<CheckCircle />
						<AlertDescription>Renamed.</AlertDescription>
					</Alert>
				)}
				{rename.error && (
					<Alert variant="danger">
						<AlertCircle />
						<AlertDescription>{rename.error.message}</AlertDescription>
					</Alert>
				)}

				<div className="space-y-1.5">
					<Label htmlFor="org-name" className="text-xs">Name</Label>
					<Input id="org-name" {...form.register("name")} feedback={form.formState.errors.name ? "error" : undefined} />
					{form.formState.errors.name && <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>}
				</div>

				<Button type="submit" variant="solid" size="sm" isLoading={rename.isPending} disabled={!form.formState.isDirty}>
					Save
				</Button>
			</form>
		</SettingsRow>
	)
}

// ── Members ──

function MembersRow({ orgId }: { orgId: string }) {
	const { data: membersData } = useQuery(orgMembersQueryOptions(orgId))
	const members = (membersData as { members?: { id: string; email: string; role: string }[] })?.members ?? []

	return (
		<SettingsRow
			title="Members"
			description={`${members.length} member${members.length !== 1 ? "s" : ""} in this organization. Manage roles and invitations from the Members page.`}
		>
			<div className="space-y-2">
				{members.length === 0 ? (
					<p className="text-sm text-muted-foreground">No members yet.</p>
				) : (
					members.slice(0, 5).map((member) => (
						<div key={member.id} className="flex items-center justify-between rounded-md border border-border px-3 py-2">
							<span className="text-sm">{member.email}</span>
							<Badge variant={member.role === "owner" ? "primary" : "neutral"}>
								{member.role === "owner" && <Crown className="mr-1 size-3" />}
								{member.role}
							</Badge>
						</div>
					))
				)}
				{members.length > 5 && (
					<p className="text-xs text-muted-foreground">and {members.length - 5} more...</p>
				)}
				<Button variant="outline" size="xs" render={<Link to="/orgs/$orgId/members" params={{ orgId }} />}>
					Manage members
				</Button>
			</div>
		</SettingsRow>
	)
}

// ── Danger Zone ──

function DangerRow({ orgId }: { orgId: string }) {
	const navigate = useNavigate()
	const { data: org } = useQuery(orgQueryOptions(orgId))
	const orgName = (org as { name?: string })?.name ?? ""
	const qc = useQueryClient()

	const form = useForm<DeleteInput>({
		resolver: zodResolver(deleteSchema),
		defaultValues: { confirmation: "" },
	})

	const deleteOrg = useMutation({
		mutationKey: mutationKeys.orgs.delete,
		mutationFn: async () => {
			const { error } = await api().DELETE("/orgs/{orgID}", {
				params: { path: { orgID: orgId } },
			})
			if (error) throw new Error("Failed to delete organization")
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: queryKeys.orgs.all })
			navigate({ to: "/" })
		},
	})

	const confirmationMatch = form.watch("confirmation") === orgName

	return (
		<SettingsRow
			title="Delete organization"
			description="Permanently delete this organization and all its projects, secrets, tokens, and audit logs. This cannot be undone."
		>
			{deleteOrg.error && (
				<Alert variant="danger" className="mb-4">
					<AlertCircle />
					<AlertDescription>{deleteOrg.error.message}</AlertDescription>
				</Alert>
			)}

			<Dialog>
				<DialogTrigger render={<Button variant="danger" size="sm" />}>
					<Trash2 />
					Delete organization
				</DialogTrigger>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete {orgName}?</DialogTitle>
						<DialogDescription>
							This is permanent. All projects, secrets, service tokens, and audit logs will be destroyed.
						</DialogDescription>
					</DialogHeader>

					<form onSubmit={form.handleSubmit(() => deleteOrg.mutate())} className="space-y-4">
						<div className="space-y-1.5">
							<Label htmlFor="delete-confirm" className="text-xs">
								Type <span className="font-mono font-semibold">{orgName}</span> to confirm
							</Label>
							<Input id="delete-confirm" placeholder={orgName} {...form.register("confirmation")} autoComplete="off" />
						</div>

						<DialogFooter>
							<DialogClose render={<Button variant="ghost" size="sm" />}>
								Cancel
							</DialogClose>
							<Button type="submit" variant="danger" size="sm" disabled={!confirmationMatch} isLoading={deleteOrg.isPending}>
								Delete permanently
							</Button>
						</DialogFooter>
					</form>
				</DialogContent>
			</Dialog>
		</SettingsRow>
	)
}
