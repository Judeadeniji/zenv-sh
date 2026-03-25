import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { z } from "zod"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "#/components/ui/sheet"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Input } from "#/components/ui/input"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Separator } from "#/components/ui/separator"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "#/components/ui/select"
import { DataTable } from "#/components/data-table"
import { SearchInput } from "#/components/search-input"
import { CreateTokenDialog } from "#/components/create-token-dialog"
import { tokensQueryOptions, useRevokeToken, useDestroyToken } from "#/lib/queries/tokens"
import { toast } from "sonner"
import { FileKey, Plus, Trash2, AlertCircle } from "lucide-react"

const searchSchema = z.object({
	page: z.number().catch(1),
	per_page: z.number().catch(50),
	search: z.string().optional(),
	status: z.enum(["active", "revoked", "all"]),
	sort_by: z.string(),
	sort_dir: z.enum(["asc", "desc"]),
}).partial()

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/tokens")({
	validateSearch: searchSchema,
	component: TokensPage,
})

interface TokenRow {
	id: string
	name: string
	permission?: string
	environment?: string
	created_at?: string
	last_used_at?: string
	revoked_at?: string
}

function TokensPage() {
	const { projectId } = Route.useParams()
	const search = Route.useSearch()
	const navigate = useNavigate({ from: Route.fullPath })

	const { data, isLoading } = useQuery(tokensQueryOptions(projectId, search))
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
			id: "status",
			header: "Status",
			cell: ({ row }) => (
				row.original.revoked_at
					? <Badge variant="danger">Revoked</Badge>
					: <Badge variant="success">Active</Badge>
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
					{!row.original.revoked_at ? (
						<RevokeTokenButton projectId={projectId} token={row.original} />
					) : (
						<DestroyTokenButton projectId={projectId} token={row.original} />
					)}
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

			<div className="mb-4 flex items-center gap-3">
				<SearchInput
					placeholder="Search tokens..."
					value={search.search}
					onChange={(val) => {
						navigate({ search: (prev) => ({ ...prev, search: val || undefined, page: 1 }), replace: true })
					}}
				/>
				<Select
					value={search.status ?? "all"}
					onValueChange={(val) => {
						navigate({
							search: (prev) => ({
								...prev,
								status: val === "all" ? undefined : (val as "active" | "revoked" | "all"),
								page: 1,
							}),
							replace: true,
						})
					}}
				>
					<SelectTrigger className="w-32.5">
						<SelectValue placeholder="All status" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All status</SelectItem>
						<SelectItem value="active">Active</SelectItem>
						<SelectItem value="revoked">Revoked</SelectItem>
					</SelectContent>
				</Select>
			</div>

			<DataTable
				columns={columns}
				data={tokens}
				pagination={data?.meta ? {
					page: data.meta.page ?? 1,
					totalPages: data.meta.total_pages ?? 1,
					total: data.meta.total ?? 0,
					onPageChange: (p) => navigate({ search: (prev) => ({ ...prev, page: p }) })
				} : undefined}
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
						<TokenDetailSheet
							projectId={projectId}
							token={selectedToken}
							onRevoked={() => setSelectedToken(null)}
						/>
					)}
				</SheetContent>
			</Sheet>
		</div>
	)
}

function TokenDetailSheet({ projectId, token, onRevoked }: {
	projectId: string
	token: TokenRow
	onRevoked: () => void
}) {
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const revoke = useRevokeToken()
	const destroy = useDestroyToken()

	const handleRevoke = () => {
		revoke.mutate(
			{ projectId, tokenId: token.id },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Revoked ${token.name}`)
					onRevoked()
				},
				onError: (err) => toast.error(err.message || "Failed to revoke token"),
			},
		)
	}

	const handleDestroy = () => {
		destroy.mutate(
			{ projectId, tokenId: token.id },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Deleted ${token.name} permanently`)
					onRevoked()
				},
				onError: (err) => toast.error(err.message || "Failed to delete token"),
			},
		)
	}

	return (
		<div className="flex-1 overflow-y-auto space-y-4 px-6 py-4">
			<div>
				<label className="text-xs font-medium text-muted-foreground">Permission</label>
				<p className="mt-1">
					<Badge variant={token.permission === "read_write" ? "warning" : "neutral"}>
						{token.permission === "read_write" ? "Read & Write" : "Read"}
					</Badge>
				</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Environment</label>
				<p className="mt-1 text-sm">{token.environment}</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Created</label>
				<p className="mt-1 text-sm">
					{token.created_at ? new Date(token.created_at).toLocaleString() : "—"}
				</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Status</label>
				<p className="mt-1">
					{token.revoked_at
						? <Badge variant="danger">Revoked</Badge>
						: <Badge variant="success">Active</Badge>
					}
				</p>
			</div>
			<div>
				<label className="text-xs font-medium text-muted-foreground">Last Used</label>
				<p className="mt-1 text-sm">
					{token.last_used_at ? new Date(token.last_used_at).toLocaleString() : "Never"}
				</p>
			</div>

			{!token.revoked_at ? (
				<>
					<Separator />

					{/* Danger Zone — type-to-confirm revoke */}
					<div className="rounded-lg border border-destructive/30 p-4">
						<div className="flex items-center justify-between">
							<div>
								<p className="text-sm font-medium">Revoke this token</p>
								<p className="mt-0.5 text-xs text-muted-foreground">
									Applications using this token will lose access immediately.
								</p>
							</div>
							<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
								<DialogTrigger render={<Button variant="danger" size="sm">Revoke</Button>} />
								<DialogContent>
									<DialogHeader>
										<DialogTitle>Revoke {token.name}?</DialogTitle>
										<DialogDescription>
											This will immediately and permanently revoke the token. Any application using it will lose access.
										</DialogDescription>
									</DialogHeader>

									<div className="py-2">
										<label className="text-xs font-medium text-muted-foreground">
											Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{token.name}</code> to confirm
										</label>
										<Input
											className="mt-1.5"
											placeholder={token.name}
											value={confirmText}
											onChange={(e) => setConfirmText(e.target.value)}
											autoFocus
										/>
									</div>

									<DialogFooter>
										<DialogClose>
											<Button variant="ghost" size="sm" type="button">Cancel</Button>
										</DialogClose>
										<Button
											variant="danger"
											size="sm"
											disabled={confirmText !== token.name}
											isLoading={revoke.isPending}
											onClick={handleRevoke}
										>
											Revoke permanently
										</Button>
									</DialogFooter>

									{revoke.error && (
										<Alert variant="danger" className="mt-2">
											<AlertCircle />
											<AlertDescription>{revoke.error.message}</AlertDescription>
										</Alert>
									)}
								</DialogContent>
							</Dialog>
						</div>
					</div>
				</>
			) : (
				<>
					<Separator />

					{/* Danger Zone — type-to-confirm destroy */}
					<div className="rounded-lg border border-destructive/30 p-4">
						<div className="flex items-center justify-between">
							<div>
								<p className="text-sm font-medium">Delete permanently</p>
								<p className="mt-0.5 text-xs text-muted-foreground">
									Remove this revoked token from the database entirely.
								</p>
							</div>
							<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
								<DialogTrigger render={<Button variant="danger" size="sm">Delete</Button>} />
								<DialogContent>
									<DialogHeader>
										<DialogTitle>Delete {token.name}?</DialogTitle>
										<DialogDescription>
											This will permanently remove the token record from the database. This action cannot be undone.
										</DialogDescription>
									</DialogHeader>

									<div className="py-2">
										<label className="text-xs font-medium text-muted-foreground">
											Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{token.name}</code> to confirm
										</label>
										<Input
											className="mt-1.5"
											placeholder={token.name}
											value={confirmText}
											onChange={(e) => setConfirmText(e.target.value)}
											autoFocus
										/>
									</div>

									<DialogFooter>
										<DialogClose>
											<Button variant="ghost" size="sm" type="button">Cancel</Button>
										</DialogClose>
										<Button
											variant="danger"
											size="sm"
											disabled={confirmText !== token.name}
											isLoading={destroy.isPending}
											onClick={handleDestroy}
										>
											Delete permanently
										</Button>
									</DialogFooter>

									{destroy.error && (
										<Alert variant="danger" className="mt-2">
											<AlertCircle />
											<AlertDescription>{destroy.error.message}</AlertDescription>
										</Alert>
									)}
								</DialogContent>
							</Dialog>
						</div>
					</div>
				</>
			)}
		</div>
	)
}

function DestroyTokenButton({ projectId, token }: {
	projectId: string
	token: TokenRow
}) {
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const destroy = useDestroyToken()

	const handleDestroy = () => {
		destroy.mutate(
			{ tokenId: token.id, projectId },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Deleted ${token.name}`)
				},
				onError: (err: any) => toast.error(err.message || "Failed to delete token"),
			},
		)
	}

	return (
		<Dialog open={confirmOpen} onOpenChange={(v) => { setConfirmOpen(v); if (!v) { setConfirmText(""); destroy.reset() } }}>
			<DialogTrigger
				render={
					<Button
						variant="ghost"
						size="icon-sm"
						className="text-muted-foreground hover:text-destructive"
						type="button"
						onClick={(e) => e.stopPropagation()}
					/>
				}
			>
				<Trash2 className="size-3.5" />
			</DialogTrigger>
			<DialogContent onClick={(e) => e.stopPropagation()}>
				<DialogHeader>
					<DialogTitle>Delete {token.name}?</DialogTitle>
					<DialogDescription>
						This will permanently remove the token record from the database. This action cannot be undone.
					</DialogDescription>
				</DialogHeader>

				<div className="py-2">
					<label className="text-xs font-medium text-muted-foreground">
						Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{token.name}</code> to confirm
					</label>
					<Input
						className="mt-1.5"
						placeholder={token.name}
						value={confirmText}
						onChange={(e) => setConfirmText(e.target.value)}
						autoFocus
					/>
				</div>

				<DialogFooter>
					<DialogClose>
						<Button variant="ghost" size="sm" type="button">Cancel</Button>
					</DialogClose>
					<Button
						variant="danger"
						size="sm"
						disabled={confirmText !== token.name}
						isLoading={destroy.isPending}
						onClick={handleDestroy}
					>
						Delete permanently
					</Button>
				</DialogFooter>

				{destroy.error && (
					<Alert variant="danger" className="mt-2">
						<AlertCircle />
						<AlertDescription>{destroy.error.message}</AlertDescription>
					</Alert>
				)}
			</DialogContent>
		</Dialog>
	)
}

function RevokeTokenButton({ projectId, token }: {
	projectId: string
	token: TokenRow
}) {
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const revoke = useRevokeToken()

	const handleRevoke = () => {
		revoke.mutate(
			{ projectId, tokenId: token.id },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					toast.success(`Revoked ${token.name}`)
				},
				onError: (err) => toast.error(err.message || "Failed to revoke token"),
			},
		)
	}

	return (
		<Dialog open={confirmOpen} onOpenChange={(v) => { setConfirmOpen(v); if (!v) { setConfirmText(""); revoke.reset() } }}>
			<DialogTrigger
				render={
					<Button
						variant="ghost"
						size="icon-sm"
						className="text-muted-foreground hover:text-destructive"
						type="button"
						onClick={(e) => e.stopPropagation()}
					/>
				}
			>
				<Trash2 className="size-3.5" />
			</DialogTrigger>
			<DialogContent onClick={(e) => e.stopPropagation()}>
				<DialogHeader>
					<DialogTitle>Revoke {token.name}?</DialogTitle>
					<DialogDescription>
						This will immediately and permanently revoke the token. Any application using it will lose access.
					</DialogDescription>
				</DialogHeader>

				<div className="py-2">
					<label className="text-xs font-medium text-muted-foreground">
						Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{token.name}</code> to confirm
					</label>
					<Input
						className="mt-1.5"
						placeholder={token.name}
						value={confirmText}
						onChange={(e) => setConfirmText(e.target.value)}
						autoFocus
					/>
				</div>

				<DialogFooter>
					<DialogClose>
						<Button variant="ghost" size="sm" type="button">Cancel</Button>
					</DialogClose>
					<Button
						variant="danger"
						size="sm"
						disabled={confirmText !== token.name}
						isLoading={revoke.isPending}
						onClick={handleRevoke}
					>
						Revoke permanently
					</Button>
				</DialogFooter>

				{revoke.error && (
					<Alert variant="danger" className="mt-2">
						<AlertCircle />
						<AlertDescription>{revoke.error.message}</AlertDescription>
					</Alert>
				)}
			</DialogContent>
		</Dialog>
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
