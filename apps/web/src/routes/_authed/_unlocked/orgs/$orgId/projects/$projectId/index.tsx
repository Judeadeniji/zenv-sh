import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { OneTimeDisplay } from "#/components/ui/one-time-display"
import { CreateSecretDialog } from "#/components/create-secret-dialog"
import { CreateTokenDialog } from "#/components/create-token-dialog"
import { projectQueryOptions, projectStatsQueryOptions, useProjectKey } from "#/lib/queries/projects"
import { tokensQueryOptions } from "#/lib/queries/tokens"
import { auditQueryOptions } from "#/lib/queries/audit"
import { useNavStore, ENVIRONMENTS } from "#/lib/stores/nav"
import { formatRelativeTime } from "#/lib/format"
import {
	KeyRound,
	FileKey,
	Shield,
	Terminal,
	AlertCircle,
	Copy,
	Check,
	Plus,
	ChevronRight,
	ArrowUpRight,
	Eye,
	EyeOff,
} from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/")({
	component: ProjectDashboard,
})

function ProjectDashboard() {
	const { orgId, projectId } = Route.useParams()
	const { data: project } = useQuery(projectQueryOptions(projectId))
	const { data: stats } = useQuery(projectStatsQueryOptions(projectId))

	const name = (project as { name?: string })?.name ?? projectId
	const totalSecrets = stats?.total_secrets ?? 0
	const totalTokens = stats?.total_service_tokens ?? 0
	const totalAuditLogs = stats?.total_audit_logs ?? 0

	return (
		<div className="w-full">

			{/* ── Header ── */}
			<ProjectHeader projectId={projectId} name={name} />

			{/* ── Inline stat strip ── */}
			<div className="mb-8 flex items-center gap-6 border-b border-border pb-6">
				<StatPill
					label="Secrets"
					value={totalSecrets}
					to="/orgs/$orgId/projects/$projectId/secrets"
					orgId={orgId}
					projectId={projectId}
					icon={<KeyRound className="size-3" />}
				/>
				<div className="h-3 w-px bg-border" />
				<StatPill
					label="Tokens"
					value={totalTokens}
					to="/orgs/$orgId/projects/$projectId/tokens"
					orgId={orgId}
					projectId={projectId}
					icon={<FileKey className="size-3" />}
				/>
				<div className="h-3 w-px bg-border" />
				<StatPill
					label="Audit logs"
					value={totalAuditLogs}
					to="/orgs/$orgId/projects/$projectId/audit"
					orgId={orgId}
					projectId={projectId}
					icon={<Shield className="size-3" />}
				/>
			</div>

			{/* ── Environment tabs ── */}
			<EnvironmentTabs orgId={orgId} projectId={projectId} stats={stats} />

			{/* ── Main content ── */}
			<div className="grid grid-cols-2 gap-8">
				<div className="space-y-8">
					<ProjectKeySection projectId={projectId} />
					<QuickStartSection projectId={projectId} />
				</div>
				<div className="space-y-8">
					<RecentActivity orgId={orgId} projectId={projectId} />
					<TokenOverview orgId={orgId} projectId={projectId} />
				</div>
			</div>
		</div>
	)
}

/* ── Header ── */

function ProjectHeader({ projectId, name }: { projectId: string; name: string }) {
	const [copied, setCopied] = useState(false)

	const handleCopyId = () => {
		navigator.clipboard?.writeText(projectId).then(() => {
			setCopied(true)
			setTimeout(() => setCopied(false), 2000)
		})
	}

	return (
		<div className="mb-6 flex items-start justify-between">
			<div>
				<h1 className="text-xl font-semibold tracking-tight">{name}</h1>
				<button
					type="button"
					onClick={handleCopyId}
					className="group mt-1 flex items-center gap-1.5 font-mono text-[11px] text-muted-foreground transition-colors hover:text-foreground"
				>
					<span>{projectId}</span>
					{copied
						? <Check className="size-3 text-primary" />
						: <Copy className="size-3 opacity-0 transition-opacity group-hover:opacity-100" />
					}
				</button>
			</div>

			<div className="flex items-center gap-2">
				<CreateSecretDialog
					projectId={projectId}
					trigger={
						<Button variant="outline" size="sm" className="h-8 gap-1.5 border-border text-xs font-normal">
							<Plus className="size-3.5" />
							Secret
						</Button>
					}
				/>
				<CreateTokenDialog
					projectId={projectId}
					trigger={
						<Button size="sm" className="h-8 gap-1.5 text-xs font-normal">
							<Plus className="size-3.5" />
							Token
						</Button>
					}
				/>
			</div>
		</div>
	)
}

/* ── Stat pill ── */

function StatPill({
	label, value, to, orgId, projectId, icon,
}: {
	label: string
	value: number
	to: string
	orgId: string
	projectId: string
	icon: React.ReactNode
}) {
	return (
		<Link
			to={to}
			params={{ orgId, projectId }}
			className="group flex items-center gap-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
		>
			<span className="opacity-60 group-hover:opacity-100">{icon}</span>
			<span className="font-mono tabular-nums text-foreground">{value}</span>
			<span>{label}</span>
		</Link>
	)
}

/* ── Environment tabs ── */

const ENV_COLORS: Record<string, string> = {
	development: "bg-blue-500",
	staging: "bg-amber-500",
	production: "bg-emerald-500",
}

function EnvironmentTabs({
	orgId,
	projectId,
	stats,
}: {
	orgId: string
	projectId: string
	stats: any
}) {
	const activeEnv = useNavStore((s) => s.activeEnvironment)
	const setEnv = useNavStore((s) => s.setActiveEnvironment)

	return (
		<div className="mb-8">
			<div className="mb-3 text-xs font-medium uppercase tracking-widest text-muted-foreground">
				Environments
			</div>
			<div className="flex gap-px overflow-hidden rounded-lg border border-border">
				{ENVIRONMENTS.map((env) => {
					const count = stats?.secrets_by_env?.[env] ?? 0
					const isActive = env === activeEnv
					return (
						<Link
							key={env}
							to="/orgs/$orgId/projects/$projectId/secrets"
							params={{ orgId, projectId }}
							onClick={() => setEnv(env)}
							className={`group flex flex-1 items-center justify-between px-4 py-3 text-sm transition-colors hover:bg-muted/50 ${
								isActive ? "bg-muted/60" : "bg-background"
							}`}
						>
							<div className="flex items-center gap-2.5">
								<div className={`size-1.5 rounded-full ${ENV_COLORS[env] ?? "bg-muted-foreground"}`} />
								<span className={`text-sm capitalize ${isActive ? "font-medium text-foreground" : "text-muted-foreground"}`}>
									{env}
								</span>
							</div>
							<div className="flex items-center gap-2">
								<span className={`font-mono text-sm tabular-nums ${isActive ? "text-foreground" : "text-muted-foreground"}`}>
									{count}
								</span>
								<ChevronRight className={`size-3 opacity-0 transition-opacity group-hover:opacity-60 ${isActive ? "opacity-60" : ""}`} />
							</div>
						</Link>
					)
				})}
			</div>
		</div>
	)
}

/* ── Project Key ── */

function ProjectKeySection({ projectId }: { projectId: string }) {
	const [revealed, setRevealed] = useState(false)
	const { data: projectKey, error, isLoading } = useProjectKey(projectId)
	const [envCopied, setEnvCopied] = useState(false)

	const handleCopyEnvLine = () => {
		if (!projectKey) return
		navigator.clipboard?.writeText(`ZENV_PROJECT_KEY=${projectKey}`).then(() => {
			setEnvCopied(true)
			setTimeout(() => setEnvCopied(false), 2000)
		})
	}

	return (
		<section>
			<SectionHeader label="Project key" />
			<div className="overflow-hidden rounded-lg border border-border">
				<div className="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-2.5">
					<span className="font-mono text-[11px] text-muted-foreground">ZENV_PROJECT_KEY</span>
					<div className="flex items-center gap-1">
						{revealed && projectKey && (
							<button
								type="button"
								onClick={handleCopyEnvLine}
								className="flex items-center gap-1.5 rounded px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
							>
								{envCopied ? <Check className="size-3 text-primary" /> : <Copy className="size-3" />}
								Copy env line
							</button>
						)}
						<button
							type="button"
							onClick={() => setRevealed((v) => !v)}
							className="flex items-center gap-1.5 rounded px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
						>
							{revealed ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
							{revealed ? "Hide" : "Reveal"}
						</button>
					</div>
				</div>

				<div className="px-4 py-3">
					{!revealed ? (
						<div className="flex items-center gap-2">
							<div className="flex gap-0.5">
								{Array.from({ length: 32 }).map((_, i) => (
									<div key={i} className="size-1.5 rounded-full bg-muted-foreground/20" />
								))}
							</div>
						</div>
					) : isLoading ? (
						<Spinner />
					) : error ? (
						<Alert variant="danger">
							<AlertCircle />
							<AlertDescription>{error.message}</AlertDescription>
						</Alert>
					) : projectKey ? (
						<>
							<OneTimeDisplay value={projectKey} label="ZENV_PROJECT_KEY" masked={false} />
							<p className="mt-2 text-[11px] text-muted-foreground">
								Unwrapped client-side using your private key. The server never sees this value.
							</p>
						</>
					) : null}
				</div>
			</div>
		</section>
	)
}

/* ── Quick Start ── */

function QuickStartSection({ projectId }: { projectId: string }) {
	return (
		<section>
			<SectionHeader label="Quick start" />
			<div className="overflow-hidden rounded-lg border border-border">
				<div className="flex items-center gap-2 border-b border-border bg-muted/30 px-4 py-2.5">
					<Terminal className="size-3.5 text-muted-foreground" />
					<span className="text-[11px] font-medium text-muted-foreground">Terminal</span>
				</div>
				<div className="divide-y divide-border">
					<Step n={1} label="Link this project" cmd={`zenv projects init ${projectId}`} />
					<Step n={2} label="Run with secrets injected" cmd="zenv run -- npm start" />
				</div>
			</div>
		</section>
	)
}

function Step({ n, label, cmd }: { n: number; label: string; cmd: string }) {
	const [copied, setCopied] = useState(false)

	const handleCopy = () => {
		navigator.clipboard?.writeText(cmd).then(() => {
			setCopied(true)
			setTimeout(() => setCopied(false), 2000)
		})
	}

	return (
		<div className="flex items-center gap-4 px-4 py-3">
			<span className="flex size-5 shrink-0 items-center justify-center rounded-full border border-border text-[10px] font-medium tabular-nums text-muted-foreground">
				{n}
			</span>
			<div className="min-w-0 flex-1">
				<p className="mb-1 text-[11px] text-muted-foreground">{label}</p>
				<code className="block truncate font-mono text-xs text-foreground">$ {cmd}</code>
			</div>
			<button
				type="button"
				onClick={handleCopy}
				className="shrink-0 rounded p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
			>
				{copied ? <Check className="size-3.5 text-primary" /> : <Copy className="size-3.5" />}
			</button>
		</div>
	)
}

/* ── Recent Activity ── */

const ACTION_RESULT_DOT: Record<string, string> = {
	success: "bg-emerald-500",
	denied: "bg-red-500",
	error: "bg-amber-500",
}

function RecentActivity({ orgId, projectId }: { orgId: string; projectId: string }) {
	const { data, isLoading } = useQuery(auditQueryOptions(projectId, { per_page: 5 }))
	const logs = data?.entries ?? []

	return (
		<section>
			<div className="mb-3 flex items-center justify-between">
				<SectionHeader label="Recent activity" noMargin />
				<Link
					to="/orgs/$orgId/projects/$projectId/audit"
					params={{ orgId, projectId }}
					className="flex items-center gap-0.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
				>
					View all <ArrowUpRight className="size-3" />
				</Link>
			</div>

			<div className="overflow-hidden rounded-lg border border-border">
				{isLoading ? (
					<div className="flex justify-center py-8"><Spinner /></div>
				) : logs.length === 0 ? (
					<p className="py-10 text-center text-xs text-muted-foreground">No activity yet</p>
				) : (
					logs.map((log, i) => (
						<div
							key={log.id}
							className={`flex items-center gap-3 px-4 py-2.5 ${i < logs.length - 1 ? "border-b border-border" : ""}`}
						>
							<div className={`size-1.5 shrink-0 rounded-full ${ACTION_RESULT_DOT[log.result!] ?? "bg-muted-foreground"}`} />
							<code className="flex-1 truncate font-mono text-xs text-foreground">{log.action}</code>
							<span className="shrink-0 tabular-nums text-[11px] text-muted-foreground">
								{log.created_at ? formatRelativeTime(log.created_at) : "—"}
							</span>
						</div>
					))
				)}
			</div>
		</section>
	)
}

/* ── Token Overview ── */

function TokenOverview({ orgId, projectId }: { orgId: string; projectId: string }) {
	const { data, isLoading } = useQuery(tokensQueryOptions(projectId, { per_page: 5 }))
	const tokens: {
		id: string
		name?: string
		permission?: string
		environment?: string
		last_used_at?: string
	}[] = (data as any)?.tokens ?? []

	return (
		<section>
			<div className="mb-3 flex items-center justify-between">
				<SectionHeader label="Service tokens" noMargin />
				<Link
					to="/orgs/$orgId/projects/$projectId/tokens"
					params={{ orgId, projectId }}
					search={{ status: "all" }}
					className="flex items-center gap-0.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
				>
					View all <ArrowUpRight className="size-3" />
				</Link>
			</div>

			<div className="overflow-hidden rounded-lg border border-border">
				{isLoading ? (
					<div className="flex justify-center py-8"><Spinner /></div>
				) : tokens.length === 0 ? (
					<p className="py-10 text-center text-xs text-muted-foreground">No tokens yet</p>
				) : (
					tokens.map((token, i) => (
						<div
							key={token.id}
							className={`flex items-center gap-3 px-4 py-2.5 ${i < tokens.length - 1 ? "border-b border-border" : ""}`}
						>
							<FileKey className="size-3.5 shrink-0 text-muted-foreground" />
							<span className="flex-1 truncate text-sm">{token.name}</span>
							<div className="flex items-center gap-1.5">
								<EnvDot env={token.environment} />
								<span className="text-[11px] capitalize text-muted-foreground">{token.environment}</span>
							</div>
							<span className="rounded border border-border px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
								{token.permission === "read_write" ? "rw" : "r"}
							</span>
						</div>
					))
				)}
			</div>
		</section>
	)
}

/* ── Shared primitives ── */

function SectionHeader({ label, noMargin }: { label: string; noMargin?: boolean }) {
	return (
		<h2 className={`text-xs font-medium uppercase tracking-widest text-muted-foreground ${noMargin ? "" : "mb-3"}`}>
			{label}
		</h2>
	)
}

function EnvDot({ env }: { env?: string }) {
	return (
		<div className={`size-1.5 rounded-full ${ENV_COLORS[env ?? ""] ?? "bg-muted-foreground/40"}`} />
	)
}