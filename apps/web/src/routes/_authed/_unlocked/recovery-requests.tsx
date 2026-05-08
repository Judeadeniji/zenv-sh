import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { type ColumnDef } from "@tanstack/react-table"
import { api } from "#/lib/api-client"
import { queryKeys, mutationKeys } from "#/lib/keys"
import { fromBase64 } from "#/lib/encoding"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "#/components/ui/dialog"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "#/components/ui/sheet"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "#/components/ui/select"
import { Textarea } from "#/components/ui/textarea"
import { DataTable } from "#/components/data-table"
import { SearchInput } from "#/components/search-input"
import { formatDateTime } from "#/lib/format"
import { AlertCircle, Users, CheckCircle2, Eye } from "lucide-react"
import { toast } from "sonner"

const approveSchema = z.object({
	recovery_payload: z
		.string()
		.min(1, "Recovery payload is required")
		.refine((s) => {
			try {
				fromBase64(s)
				return true
			} catch {
				return false
			}
		}, "Must be valid base64"),
})
type ApproveInput = z.infer<typeof approveSchema>

interface IncomingRequest {
	request_id: string
	requester_name: string
	requester_email: string
	status: "pending" | "approved" | string
	eligible_at: string
	requested_at: string
}

export const Route = createFileRoute("/_authed/_unlocked/recovery-requests")({
	component: RecoveryRequestsPage,
})

function RecoveryRequestsPage() {
	const qc = useQueryClient()
	const [search, setSearch] = useState("")
	const [status, setStatus] = useState<"all" | "pending" | "approved">("all")
	const [selected, setSelected] = useState<IncomingRequest | null>(null)

	const incoming = useQuery({
		queryKey: queryKeys.recovery.incomingRequests,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/incoming-requests")
			if (error) throw new Error("Failed to fetch incoming requests")
			return (data ?? []) as IncomingRequest[]
		},
		refetchInterval: 30_000,
	})

	const approve = useMutation({
		mutationKey: mutationKeys.recovery.approve,
		mutationFn: async ({ requestId, recoveryPayload }: { requestId: string; recoveryPayload: string }) => {
			const { error } = await api().POST(
				"/auth/recovery/request/{id}/approve",
				{
					params: { path: { id: requestId } },
					body: { recovery_payload: recoveryPayload },
				},
			)
			if (error) throw new Error(error.error || "Failed to approve request")
		},
		onSuccess: async () => {
			await qc.invalidateQueries({ queryKey: queryKeys.recovery.incomingRequests })
			toast.success("Request approved successfully")
		},
		onError: (error) => toast.error(error.message),
	})

	const rows = (incoming.data ?? [])
		.filter((r) => (status === "all" ? true : r.status === status))
		.filter((r) => {
			if (!search.trim()) return true
			return r.requester_email.toLowerCase().includes(search.trim().toLowerCase())
		})

	const columns: ColumnDef<IncomingRequest, unknown>[] = [
		{
			accessorKey: "requester_name",
			header: "Requester",
			cell: ({ row }) => (
				<div className="min-w-0">
					<p className="truncate text-sm font-medium">{row.original.requester_name || row.original.requester_email}</p>
					<p className="text-xs text-muted-foreground">ID: {row.original.request_id.slice(0, 8)}…</p>
				</div>
			),
		},
		{
			accessorKey: "status",
			header: "Status",
			cell: ({ row }) => {
				const eligibleAt = new Date(row.original.eligible_at)
				const isEligible = Date.now() >= eligibleAt.getTime()
				return (
					<Badge
						variant={row.original.status === "approved" ? "success" : isEligible ? "warning" : "neutral"}
					>
						{row.original.status === "approved" ? "Approved" : isEligible ? "Eligible" : "Waiting"}
					</Badge>
				)
			},
		},
		{
			accessorKey: "requested_at",
			header: "Requested",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{formatDateTime(row.original.requested_at)}
				</span>
			),
		},
		{
			accessorKey: "eligible_at",
			header: "Eligible",
			cell: ({ row }) => (
				<span className="text-xs text-muted-foreground">
					{formatDateTime(row.original.eligible_at)}
				</span>
			),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) => (
				<div className="text-right">
					<Button
						variant="outline"
						size="xs"
						title="View recovery request"
						onClick={(e) => {
							e.stopPropagation()
							setSelected(row.original)
						}}
					>
						<Eye />
					</Button>
				</div>
			),
		},
	]

	return (
		<div>
			<div className="mb-6 flex items-center justify-between">
				<div>
					<h1 className="text-lg font-semibold">Recovery Requests</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						Incoming requests where you’re the trusted contact.
					</p>
				</div>
				<Button variant="outline" size="sm" render={<Link to="/recover/trusted-contact" />}>
					Request recovery (for your account)
				</Button>
			</div>

			<div className="mb-4 flex items-center gap-3">
				<SearchInput
					placeholder="Search by requester email..."
					value={search}
					onChange={(val) => setSearch(val)}
				/>
				<Select value={status} onValueChange={(v) => setStatus(v as typeof status)}>
					<SelectTrigger className="w-32.5">
						<SelectValue placeholder="All status" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All status</SelectItem>
						<SelectItem value="pending">Pending</SelectItem>
						<SelectItem value="approved">Approved</SelectItem>
					</SelectContent>
				</Select>
			</div>

			{incoming.error && (
				<Alert variant="danger" className="mb-4">
					<AlertCircle />
					<AlertDescription>{incoming.error.message}</AlertDescription>
				</Alert>
			)}

			<DataTable
				columns={columns}
				data={rows}
				onRowClick={(row) => setSelected(row.original)}
				emptyIcon={<Users />}
				emptyTitle="No incoming requests"
				emptyDescription="If you’re a trusted contact, requests will show up here when someone initiates recovery."
			/>

			<Sheet open={!!selected} onOpenChange={(open) => { if (!open) setSelected(null) }}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>{selected?.requester_email ?? "Recovery request"}</SheetTitle>
						<SheetDescription>Review request details and approve when eligible.</SheetDescription>
					</SheetHeader>
					{selected && (
						<RequestDetailSheet
							request={selected}
							approvePending={approve.isPending}
							approveError={approve.error?.message}
							onApprove={(payload) =>
								approve.mutate(
									{ requestId: selected.request_id, recoveryPayload: payload },
									{
										onSuccess: async () => {
											setSelected(null)
											await qc.invalidateQueries({ queryKey: queryKeys.recovery.incomingRequests })
											toast.success("Request approved successfully")
										},
										onError: (error) => toast.error(error.message),
									},
								)
							}
						/>
					)}
				</SheetContent>
			</Sheet>
		</div>
	)
}

function RequestDetailSheet({
	request,
	approvePending,
	approveError,
	onApprove,
}: {
	request: IncomingRequest
	approvePending: boolean
	approveError?: string
	onApprove: (payloadBase64: string) => void
}) {
	const eligibleAt = new Date(request.eligible_at)
	const isEligible = Date.now() >= eligibleAt.getTime()
	const canApprove = request.status === "pending" && isEligible

	const form = useForm<ApproveInput>({
		resolver: zodResolver(approveSchema),
		defaultValues: { recovery_payload: "" },
	})

	return (
		<div className="flex-1 overflow-y-auto space-y-4 px-6 py-4">
			<div>
				<label className="text-xs font-medium text-muted-foreground">Status</label>
				<p className="mt-1">
					<Badge variant={request.status === "approved" ? "success" : canApprove ? "warning" : "neutral"}>
						{request.status === "approved" ? "Approved" : canApprove ? "Eligible" : "Pending"}
					</Badge>
				</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Requested</label>
				<p className="mt-1 text-sm">{formatDateTime(request.requested_at)}</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Eligible</label>
				<p className="mt-1 text-sm">{formatDateTime(request.eligible_at)}</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Request ID</label>
				<p className="mt-1 font-mono text-xs text-muted-foreground">{request.request_id}</p>
			</div>

			{request.status === "approved" ? (
				<Alert variant="success">
					<CheckCircle2 />
					<AlertDescription>
						Approved. The requester can now complete recovery.
					</AlertDescription>
				</Alert>
			) : (
				<div className="pt-2">
					<Dialog
						onOpenChange={(open) => {
							if (!open) form.reset({ recovery_payload: "" })
						}}
					>
						<DialogTrigger
							render={<Button variant="solid" size="sm" disabled={!canApprove} />}
						>
							Approve request
						</DialogTrigger>
						<DialogContent>
							<DialogHeader>
								<DialogTitle>Approve recovery request</DialogTitle>
								<DialogDescription>
									Paste the base64 <code>recovery_payload</code> for this request.
								</DialogDescription>
							</DialogHeader>

							<form
								onSubmit={form.handleSubmit((d) => onApprove(d.recovery_payload))}
								className="space-y-3"
							>
								<div className="space-y-1.5">
									<label className="text-xs font-medium text-muted-foreground">
										Recovery payload (base64)
									</label>
									<Textarea
										rows={6}
										placeholder="base64..."
										{...form.register("recovery_payload")}
									/>
									{form.formState.errors.recovery_payload && (
										<p className="text-xs text-destructive">{form.formState.errors.recovery_payload.message}</p>
									)}
								</div>

								{approveError && (
									<Alert variant="danger">
										<AlertCircle />
										<AlertDescription>{approveError}</AlertDescription>
									</Alert>
								)}

								<DialogFooter>
									<Button type="submit" variant="solid" isLoading={approvePending}>
										Approve
									</Button>
									<DialogClose render={<Button type="button" variant="ghost" />}>
										Close
									</DialogClose>
								</DialogFooter>
							</form>
						</DialogContent>
					</Dialog>
					<p className="mt-2 text-xs text-muted-foreground">
						Approve is enabled only after the 72-hour waiting period.
					</p>
				</div>
			)}
		</div>
	)
}

