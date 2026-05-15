import { useQuery } from "@tanstack/react-query"
import { Link, useMatches, useParams } from "@tanstack/react-router"
import { Loader2 } from "lucide-react"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "#/components/ui/breadcrumb"
import { Button } from "#/components/ui/button"
import { Separator } from "#/components/ui/separator"
import { SidebarTrigger } from "#/components/ui/sidebar"
import { authClient } from "#/lib/auth-client"
import { useUpdatePreferences } from "#/lib/queries/preferences"
import { projectsQueryOptions } from "#/lib/queries/projects"
import { useAuthStore } from "#/lib/stores/auth"
import { ENVIRONMENTS, useNavStore } from "#/lib/stores/nav"

function queryErrorMessage(err: { message?: string } | null | undefined) {
	if (!err) return "Something went wrong"
	return err.message?.trim() || "Something went wrong"
}

export function AppHeader() {
	const params = useParams({ strict: false }) as { orgId?: string; projectId?: string }
	const matches = useMatches()
	const crypto = useAuthStore((s) => s.crypto)

	const {
		data: activeOrg,
		isPending: isOrgPending,
		isRefetching: isOrgRefetching,
		error: activeOrgError,
		refetch: refetchActiveOrg,
	} = authClient.useActiveOrganization()

	const orgId = activeOrg?.id

	const {
		data: projectsData,
		isPending: isProjectsPending,
		isFetching: isProjectsFetching,
		error: projectsError,
		refetch: refetchProjects,
	} = useQuery({
		...projectsQueryOptions(orgId ?? ""),
		enabled: !!orgId && !!crypto,
	})
	const projectList = projectsData?.projects ?? []
	const activeProject = projectList.find((p) => p.id === params.projectId)

	const projectLinkParams =
		orgId && activeProject?.id ? { orgId, projectId: activeProject.id } : null

	const showProjectCrumb = !!params.projectId && !!orgId && !!crypto
	const projectsLoading = showProjectCrumb && isProjectsPending && !activeProject
	const projectsFailed = showProjectCrumb && !!projectsError

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
					{isOrgPending && !activeOrg && !activeOrgError ? (
						<BreadcrumbItem>
							<span className="inline-flex items-center gap-1.5 text-muted-foreground text-sm">
								<Loader2 className="size-3.5 animate-spin" aria-hidden />
								<span className="sr-only">Loading organization</span>
								<span
									aria-hidden
									className="max-w-40 truncate rounded bg-muted px-2 py-0.5 font-normal"
								>
									Organization
								</span>
							</span>
						</BreadcrumbItem>
					) : null}
					{activeOrgError ? (
						<BreadcrumbItem>
							<span className="flex flex-wrap items-center gap-2 text-destructive text-sm">
								<span
									className="max-w-[min(18rem,50vw)] truncate"
									title={queryErrorMessage(activeOrgError)}
								>
									{queryErrorMessage(activeOrgError)}
								</span>
								<Button
									type="button"
									variant="outline"
									size="sm"
									className="h-7 shrink-0 px-2 text-xs"
									disabled={isOrgRefetching}
									onClick={() => {
										void refetchActiveOrg()
									}}
								>
									{isOrgRefetching ? (
										<>
											<Loader2 className="mr-1 size-3 animate-spin" aria-hidden />
											Retrying
										</>
									) : (
										"Retry"
									)}
								</Button>
							</span>
						</BreadcrumbItem>
					) : null}
					{orgId && activeOrg && !activeOrgError ? (
						<BreadcrumbItem>
							<BreadcrumbLink render={(p) => <Link {...p} to="/orgs/$orgId" params={{ orgId }} />}>
								{activeOrg.name}
							</BreadcrumbLink>
						</BreadcrumbItem>
					) : null}
					{showProjectCrumb ? (
						<>
							<BreadcrumbSeparator />
							<BreadcrumbItem>
								{projectsFailed ? (
									<span className="flex flex-wrap items-center gap-2 text-destructive text-sm">
										<span
											className="max-w-[min(18rem,50vw)] truncate"
											title={queryErrorMessage(projectsError)}
										>
											{queryErrorMessage(projectsError)}
										</span>
										<Button
											type="button"
											variant="outline"
											size="sm"
											className="h-7 shrink-0 px-2 text-xs"
											disabled={isProjectsFetching}
											onClick={() => {
												void refetchProjects()
											}}
										>
											{isProjectsFetching ? (
												<>
													<Loader2 className="mr-1 size-3 animate-spin" aria-hidden />
													Retrying
												</>
											) : (
												"Retry"
											)}
										</Button>
									</span>
								) : projectsLoading ? (
									<span className="inline-flex items-center gap-1.5 text-muted-foreground text-sm">
										<Loader2 className="size-3.5 animate-spin" aria-hidden />
										<span className="sr-only">Loading project</span>
										<span
											aria-hidden
											className="max-w-40 truncate rounded bg-muted px-2 py-0.5 font-normal"
										>
											Project
										</span>
									</span>
								) : activeProject ? (
									section && projectLinkParams ? (
										<BreadcrumbLink
											render={(p) => (
												<Link
													{...p}
													to="/orgs/$orgId/projects/$projectId"
													params={projectLinkParams}
												/>
											)}
										>
											{activeProject.name}
										</BreadcrumbLink>
									) : (
										<BreadcrumbPage>{activeProject.name}</BreadcrumbPage>
									)
								) : (
									<BreadcrumbPage className="text-muted-foreground">Project</BreadcrumbPage>
								)}
							</BreadcrumbItem>
						</>
					) : null}
					{section ? (
						<>
							<BreadcrumbSeparator />
							<BreadcrumbItem>
								<BreadcrumbPage>{section}</BreadcrumbPage>
							</BreadcrumbItem>
						</>
					) : null}
				</BreadcrumbList>
			</Breadcrumb>

			<div className="ml-auto flex items-center gap-2">{params.projectId && <EnvSwitcher />}</div>
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
	const activeEnv = useNavStore((s) => s.activeEnvironment)
	const setActiveEnv = useNavStore((s) => s.setActiveEnvironment)
	const updatePrefs = useUpdatePreferences()

	const handleChange = (env: string) => {
		setActiveEnv(env)
		updatePrefs.mutate({ active_environment: env })
	}

	return (
		<div className="flex items-center rounded-md bg-muted p-0.5">
			{ENVIRONMENTS.map((env) => {
				const isActive = env === activeEnv
				return (
					<button
						key={env}
						type="button"
						onClick={() => handleChange(env)}
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
