import { SidebarTrigger } from "#/components/ui/sidebar"
import { Separator } from "#/components/ui/separator"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "#/components/ui/breadcrumb"
import { useQuery } from "@tanstack/react-query"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { projectsQueryOptions } from "#/lib/queries/projects"
import { useNavStore, ENVIRONMENTS } from "#/lib/stores/nav"
import { Link, useParams, useMatches } from "@tanstack/react-router"

export function AppHeader() {
	const params = useParams({ strict: false }) as { orgId?: string; projectId?: string }
	const matches = useMatches()

	const { data: orgsData } = useQuery(orgsQueryOptions)
	const orgList = (orgsData as { organizations?: { id: string; name: string }[] })?.organizations ?? []
	const activeOrg = orgList.find((o) => o.id === params.orgId)

	const { data: projectsData } = useQuery({
		...projectsQueryOptions(activeOrg?.id ?? ""),
		enabled: !!activeOrg,
	})
	const projectList = (projectsData as { projects?: { id: string; name: string }[] })?.projects ?? []
	const activeProject = projectList.find((p) => p.id === params.projectId)

	// Derive current section from the last route match
	const lastMatch = matches[matches.length - 1]
	const routeId = lastMatch?.routeId ?? ""
	const section = routeId.includes("/secrets")
		? "Secrets"
		: routeId.includes("/tokens")
			? "Tokens"
			: routeId.includes("/audit")
				? "Audit Log"
				: routeId.includes("/members")
					? "Members"
					: routeId.includes("/settings") && params.orgId
						? "Org Settings"
						: routeId.includes("/settings")
							? "Settings"
							: null

	return (
		<header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-4">
			<SidebarTrigger className="-ml-1" />
			<Separator orientation="vertical" className="mr-2 h-4!" />

			<Breadcrumb>
				<BreadcrumbList>
					{activeOrg && (
						<BreadcrumbItem>
							<BreadcrumbLink
								render={(p) => (
									<Link
										{...p}
										to="/orgs/$orgId"
										params={{ orgId: activeOrg.id }}
									/>
								)}
							>
								{activeOrg.name}
							</BreadcrumbLink>
						</BreadcrumbItem>
					)}
					{activeProject && (
						<>
							<BreadcrumbSeparator />
							<BreadcrumbItem>
								{section ? (
									<BreadcrumbLink
										render={(p) => (
											<Link
												{...p}
												to="/orgs/$orgId/projects/$projectId"
												params={{ orgId: activeOrg!.id, projectId: activeProject.id }}
											/>
										)}
									>
										{activeProject.name}
									</BreadcrumbLink>
								) : (
									<BreadcrumbPage>{activeProject.name}</BreadcrumbPage>
								)}
							</BreadcrumbItem>
						</>
					)}
					{section && (
						<>
							<BreadcrumbSeparator />
							<BreadcrumbItem>
								<BreadcrumbPage>{section}</BreadcrumbPage>
							</BreadcrumbItem>
						</>
					)}
				</BreadcrumbList>
			</Breadcrumb>

			<div className="ml-auto flex items-center gap-2">
				{params.projectId && <EnvSwitcher />}
			</div>
		</header>
	)
}

const ENV_SHORT: Record<string, string> = {
	development: "Dev",
	staging: "Stg",
	production: "Prod",
}

const ENV_DOT: Record<string, string> = {
	development: "bg-blue-500",
	staging: "bg-amber-500",
	production: "bg-emerald-500",
}

function EnvSwitcher() {
	const active = useNavStore((s) => s.activeEnvironment)
	const setEnv = useNavStore((s) => s.setActiveEnvironment)

	return (
		<div className="flex items-center rounded-md bg-muted p-0.5">
			{ENVIRONMENTS.map((env) => {
				const isActive = env === active
				return (
					<button
						key={env}
						type="button"
						onClick={() => setEnv(env)}
						className={`flex items-center gap-1.5 rounded-[5px] px-2.5 py-1 text-xs font-medium transition-all ${
							isActive
								? "bg-background text-foreground shadow-sm"
								: "text-muted-foreground hover:text-foreground"
						}`}
					>
						<span className={`size-1.5 rounded-full ${ENV_DOT[env] ?? "bg-muted-foreground"}`} />
						{ENV_SHORT[env] ?? env}
					</button>
				)
			})}
		</div>
	)
}
