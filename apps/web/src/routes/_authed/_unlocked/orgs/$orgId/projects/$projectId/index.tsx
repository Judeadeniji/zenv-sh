import { useState } from "react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Spinner } from "#/components/ui/spinner"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { OneTimeDisplay } from "#/components/ui/one-time-display"
import { CreateSecretDialog } from "#/components/create-secret-dialog"
import { CreateTokenDialog } from "#/components/create-token-dialog"
import { projectQueryOptions, useProjectKey } from "#/lib/queries/projects"
import { secretsQueryOptions } from "#/lib/queries/secrets"
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
	ArrowRight,
} from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/")({
	component: ProjectDashboard,
})

function ProjectDashboard() {
	const { orgId, projectId } = Route.useParams()
	const { data: project } = useQuery(projectQueryOptions(projectId))

	const name = (project as { name?: string })?.name ?? projectId

	return (
		<div>
			<DashboardHeader projectId={projectId} name={name} />
			<EnvironmentBreakdown orgId={orgId} projectId={projectId} />
			<StatsGrid orgId={orgId} projectId={projectId} />

			<div className="grid gap-4 lg:grid-cols-2">
				<div className="space-y-4">
					<ProjectKeySection projectId={projectId} />
					<QuickStartSection projectId={projectId} />
				</div>
				<div className="space-y-4">
					<RecentActivity orgId={orgId} projectId={projectId} />
					<TokenOverview orgId={orgId} projectId={projectId} />
				</div>
			</div>
		</div>
	)
}

/* ── Header ── */

function DashboardHeader({ projectId, name }: { projectId: string; name: string }) {
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
				<h1 className="text-lg font-semibold">{name}</h1>
				<button
					type="button"
					onClick={handleCopyId}
					className="mt-0.5 flex items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground"
				>
					{projectId}
					{copied ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
				</button>
			</div>
			<div className="flex gap-2">
				<CreateSecretDialog
					projectId={projectId}
					trigger={<Button type="button" variant="outline" size="sm"><Plus /> Secret</Button>}
				/>
				<CreateTokenDialog
					projectId={projectId}
					trigger={<Button type="button" size="sm"><Plus /> Token</Button>}
				/>
			</div>
		</div>
	)
}

/* ── Environment Breakdown ── */

function EnvironmentBreakdown({ orgId, projectId }: { orgId: string; projectId: string }) {
	const activeEnv = useNavStore((s) => s.activeEnvironment)
	const setEnv = useNavStore((s) => s.setActiveEnvironment)

	const envQueries = ENVIRONMENTS.map((env) => ({
		env,
		// eslint-disable-next-line react-hooks/rules-of-hooks
		query: useQuery({ ...secretsQueryOptions(projectId, env), staleTime: 30_000 }),
	}))

	const envColors: Record<string, string> = {
		development: "bg-blue-500",
		staging: "bg-amber-500",
		production: "bg-emerald-500",
	}

	return (
		<div className="mb-4 grid gap-3 sm:grid-cols-3">
			{envQueries.map(({ env, query }) => {
				const count = ((query.data as { secrets?: unknown[] })?.secrets ?? []).length
				const isActive = env === activeEnv
				return (
					<Link
						key={env}
						to="/orgs/$orgId/projects/$projectId/secrets"
						params={{ orgId, projectId }}
						onClick={() => setEnv(env)}
						className={`group rounded-lg border p-3 transition-colors hover:bg-muted/50 ${isActive ? "border-primary/40 bg-primary/5" : "border-border"}`}
					>
						<div className="flex items-center gap-2">
							<div className={`size-2 rounded-full ${envColors[env] ?? "bg-muted-foreground"}`} />
							<span className="text-xs font-medium capitalize">{env}</span>
							{isActive && <Badge variant="primary" className="ml-auto text-[10px]">active</Badge>}
						</div>
						<p className="mt-2 text-xl font-semibold tabular-nums">
							{query.isLoading ? <Spinner className="size-4" /> : count}
						</p>
						<p className="text-[11px] text-muted-foreground">secrets</p>
					</Link>
				)
			})}
		</div>
	)
}

/* ── Stat Cards ── */

function StatsGrid({ orgId, projectId }: { orgId: string; projectId: string }) {
	const { data: tokensData } = useQuery(tokensQueryOptions(projectId))
	const tokenList = (tokensData as { tokens?: unknown[] })?.tokens ?? []

	// Sum secrets across all envs (use the env breakdown queries via cache)
	const devQ = useQuery({ ...secretsQueryOptions(projectId, "development"), staleTime: 30_000 })
	const stgQ = useQuery({ ...secretsQueryOptions(projectId, "staging"), staleTime: 30_000 })
	const prdQ = useQuery({ ...secretsQueryOptions(projectId, "production"), staleTime: 30_000 })

	const totalSecrets =
		((devQ.data as { secrets?: unknown[] })?.secrets ?? []).length +
		((stgQ.data as { secrets?: unknown[] })?.secrets ?? []).length +
		((prdQ.data as { secrets?: unknown[] })?.secrets ?? []).length

	return (
		<div className="mb-6 grid gap-4 sm:grid-cols-3">
			<Link
				to="/orgs/$orgId/projects/$projectId/secrets"
				params={{ orgId, projectId }}
				className="group rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
			>
				<div className="flex items-center gap-3">
					<div className="flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-foreground">
						<KeyRound className="size-4" />
					</div>
					<div>
						<p className="text-2xl font-semibold tabular-nums">{totalSecrets}</p>
						<p className="text-xs text-muted-foreground">Total secrets</p>
					</div>
				</div>
			</Link>

			<Link
				to="/orgs/$orgId/projects/$projectId/tokens"
				params={{ orgId, projectId }}
				className="group rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
			>
				<div className="flex items-center gap-3">
					<div className="flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-foreground">
						<FileKey className="size-4" />
					</div>
					<div>
						<p className="text-2xl font-semibold tabular-nums">{tokenList.length}</p>
						<p className="text-xs text-muted-foreground">Service tokens</p>
					</div>
				</div>
			</Link>

			<Link
				to="/orgs/$orgId/projects/$projectId/audit"
				params={{ orgId, projectId }}
				className="group rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
			>
				<div className="flex items-center gap-3">
					<div className="flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-foreground">
						<Shield className="size-4" />
					</div>
					<div>
						<p className="text-2xl font-semibold tabular-nums">&mdash;</p>
						<p className="text-xs text-muted-foreground">Audit log</p>
					</div>
				</div>
			</Link>
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

	if (!revealed) {
		return (
			<section className="rounded-lg border border-border p-4">
				<div className="flex items-center justify-between">
					<div>
						<h2 className="text-sm font-medium">Project Key</h2>
						<p className="mt-0.5 text-xs text-muted-foreground">
							Set as <code className="rounded bg-muted px-1 py-0.5 text-[11px]">ZENV_PROJECT_KEY</code> for CLI/SDK access.
						</p>
					</div>
					<Button variant="outline" size="sm" onClick={() => setRevealed(true)}>
						Reveal
					</Button>
				</div>
			</section>
		)
	}

	return (
		<section className="rounded-lg border border-border p-4">
			<div className="mb-3 flex items-center justify-between">
				<h2 className="text-sm font-medium">Project Key</h2>
				{projectKey && (
					<Button variant="ghost" size="sm" className="text-xs" onClick={handleCopyEnvLine}>
						{envCopied ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
						Copy as .env
					</Button>
				)}
			</div>
			{isLoading && <Spinner />}
			{error && (
				<Alert variant="danger">
					<AlertCircle />
					<AlertDescription>{error.message}</AlertDescription>
				</Alert>
			)}
			{projectKey && (
				<>
					<OneTimeDisplay value={projectKey} label="ZENV_PROJECT_KEY" masked={false} />
					<p className="mt-2 text-[11px] text-muted-foreground">
						Unwrapped in your browser using your private key. The server never sees this value.
					</p>
				</>
			)}
		</section>
	)
}

/* ── Quick Start ── */

function QuickStartSection({ projectId }: { projectId: string }) {
	return (
		<section className="rounded-lg border border-border p-4">
			<div className="mb-3 flex items-center gap-2">
				<Terminal className="size-4 text-muted-foreground" />
				<h2 className="text-sm font-medium">Quick Start</h2>
			</div>
			<div className="space-y-3">
				<Step n={1} label="Link this project" cmd={`zenv projects init ${projectId}`} />
				<Step n={2} label="Run with secrets injected" cmd="zenv run -- npm start" />
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
		<div>
			<p className="mb-1 text-xs text-muted-foreground">
				<span className="mr-1.5 inline-flex size-4 items-center justify-center rounded-full bg-muted text-[10px] font-medium">
					{n}
				</span>
				{label}
			</p>
			<div className="flex items-center gap-2">
				<code className="flex-1 rounded-md bg-muted px-2.5 py-1.5 font-mono text-xs">$ {cmd}</code>
				<Button variant="outline" size="icon-sm" onClick={handleCopy}>
					{copied ? <Check className="text-success" /> : <Copy />}
				</Button>
			</div>
		</div>
	)
}

/* ── Recent Activity ── */

function RecentActivity({ orgId, projectId }: { orgId: string; projectId: string }) {
	const { data, isLoading } = useQuery(auditQueryOptions(projectId, { perPage: 5 }))
	const logs = data?.entries ?? []

	return (
		<section className="rounded-lg border border-border p-4">
			<div className="mb-3 flex items-center justify-between">
				<h2 className="text-sm font-medium">Recent Activity</h2>
				<Link
					to="/orgs/$orgId/projects/$projectId/audit"
					params={{ orgId, projectId }}
					className="text-xs text-muted-foreground hover:text-foreground"
				>
					View all <ArrowRight className="ml-0.5 inline size-3" />
				</Link>
			</div>

			{isLoading ? (
				<div className="flex justify-center py-4"><Spinner /></div>
			) : logs.length === 0 ? (
				<p className="py-4 text-center text-xs text-muted-foreground">No activity yet</p>
			) : (
				<div className="space-y-2">
					{logs.map((log) => (
						<div key={log.id} className="flex items-center gap-3">
							<Badge
								variant={log.result === "success" ? "success" : log.result === "denied" ? "danger" : "neutral"}
								className="text-[10px]"
							>
								{log.action}
							</Badge>
							<span className="ml-auto text-[11px] text-muted-foreground">
								{log.created_at ? formatRelativeTime(log.created_at) : "—"}
							</span>
						</div>
					))}
				</div>
			)}
		</section>
	)
}

/* ── Token Overview ── */

function TokenOverview({ orgId, projectId }: { orgId: string; projectId: string }) {
	const { data, isLoading } = useQuery(tokensQueryOptions(projectId))
	const tokens: { id: string; name?: string; permission?: string; environment?: string; last_used_at?: string }[] =
		(data as { tokens?: { id: string; name?: string; permission?: string; environment?: string; last_used_at?: string }[] })?.tokens ?? []

	const display = tokens.slice(0, 5)

	return (
		<section className="rounded-lg border border-border p-4">
			<div className="mb-3 flex items-center justify-between">
				<h2 className="text-sm font-medium">Service Tokens</h2>
				<Link
					to="/orgs/$orgId/projects/$projectId/tokens"
					params={{ orgId, projectId }}
					className="text-xs text-muted-foreground hover:text-foreground"
				>
					View all <ArrowRight className="ml-0.5 inline size-3" />
				</Link>
			</div>

			{isLoading ? (
				<div className="flex justify-center py-4"><Spinner /></div>
			) : display.length === 0 ? (
				<p className="py-4 text-center text-xs text-muted-foreground">No tokens yet</p>
			) : (
				<div className="space-y-2">
					{display.map((token) => (
						<div key={token.id} className="flex items-center gap-2">
							<FileKey className="size-3.5 text-muted-foreground" />
							<span className="flex-1 truncate text-sm font-medium">{token.name}</span>
							<Badge variant="neutral" className="text-[10px]">{token.permission === "read_write" ? "rw" : "r"}</Badge>
							<Badge variant="neutral" className="text-[10px]">{token.environment}</Badge>
						</div>
					))}
					{tokens.length > 5 && (
						<p className="text-xs text-muted-foreground">+{tokens.length - 5} more</p>
					)}
				</div>
			)}
		</section>
	)
}
