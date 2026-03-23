import { Link, useNavigate, useParams } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { projectsQueryOptions } from "#/lib/queries/projects"
import { meQueryOptions } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"
import { authClient } from "#/lib/auth-client"
import { getProjectItems, getOrgItems } from "#/lib/nav-items"
import { CreateProjectDialog } from "#/components/create-project-dialog"
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupLabel,
	SidebarGroupContent,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarMenuSkeleton,
	useSidebar,
} from "#/components/ui/sidebar"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu"
import { Avatar } from "#/components/ui/avatar"
import {
	FolderKey,
	Settings,
	Lock,
	LogOut,
	Plus,
	ChevronDown,
	Building2,
	ChevronsUpDown,
} from "lucide-react"

export function AppSidebar() {
	const navigate = useNavigate()
	const { state } = useSidebar()
	const { data: me } = useQuery(meQueryOptions)
	const { data: orgsData, isLoading: orgsLoading } = useQuery(orgsQueryOptions)

	// Read orgId and projectId from URL — these may be undefined
	// when on routes like /settings or /onboarding
	const params = useParams({ strict: false }) as { orgId?: string; projectId?: string }
	const orgId = params.orgId
	const projectId = params.projectId

	const orgList = (orgsData as { organizations?: { id: string; name: string }[] })?.organizations ?? []
	const activeOrg = orgList.find((o) => o.id === orgId) ?? orgList[0]

	const { data: projectsData, isLoading: projectsLoading } = useQuery({
		...projectsQueryOptions(activeOrg?.id ?? ""),
		enabled: !!activeOrg,
	})
	const projectList = (projectsData as { projects?: { id: string; name: string }[] })?.projects ?? []

	const initials = me?.email?.slice(0, 2).toUpperCase() ?? "?"

	// Build nav items based on current context
	const manageItems = [
		...(activeOrg && projectId ? getProjectItems(activeOrg.id, projectId) : []),
		...(activeOrg ? getOrgItems(activeOrg.id) : []),
	]

	const handleLock = () => {
		useAuthStore.getState().lock()
		navigate({ to: "/unlock" })
	}

	const handleSignOut = () => {
		authClient.signOut()
		useAuthStore.getState().lock()
		navigate({ to: "/login" })
	}

	return (
		<Sidebar collapsible="icon">
			{/* ── Header: Org switcher ── */}
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<DropdownMenu>
							<DropdownMenuTrigger
								render={
									<SidebarMenuButton size="lg" className="data-open:bg-sidebar-accent">
										<div className="flex aspect-square size-7 items-center justify-center rounded-md bg-primary text-xs font-bold text-primary-foreground">
											{activeOrg?.name?.[0]?.toUpperCase() ?? "z"}
										</div>
										<div className="grid flex-1 text-left text-sm leading-tight">
											<span className="truncate font-medium">{activeOrg?.name ?? "zEnv"}</span>
											<span className="truncate text-xs text-muted-foreground">Organization</span>
										</div>
										<ChevronsUpDown className="ml-auto size-4" />
									</SidebarMenuButton>
								}
							/>
							<DropdownMenuContent className="min-w-56" align="start">
								{orgList.map((org) => (
									<DropdownMenuItem
										key={org.id}
										onClick={() => navigate({ to: "/orgs/$orgId", params: { orgId: org.id } })}
									>
										<Building2 />
										{org.name}
									</DropdownMenuItem>
								))}
								<DropdownMenuSeparator />
								<DropdownMenuItem onClick={() => navigate({ to: "/onboarding" })}>
									<Plus />
									Create organization
								</DropdownMenuItem>
							</DropdownMenuContent>
						</DropdownMenu>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>

			<SidebarContent>
				{/* ── Projects ── */}
				<SidebarGroup>
					<SidebarGroupLabel>Projects</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{projectsLoading || orgsLoading ? (
								<>
									<SidebarMenuSkeleton />
									<SidebarMenuSkeleton />
								</>
							) : projectList.length === 0 ? (
								<p
									className="px-2 py-1.5 text-xs text-muted-foreground"
									hidden={state === "collapsed"}
								>
									No projects yet
								</p>
							) : (
								projectList.map((project) => (
									<SidebarMenuItem key={project.id}>
										<SidebarMenuButton
											isActive={project.id === projectId}
											tooltip={project.name}
											render={(
												<Link
													to="/orgs/$orgId/projects/$projectId"
													params={{ orgId: activeOrg?.id ?? "", projectId: project.id! }}
												/>
											)}
										>
											<FolderKey />
											<span>{project.name}</span>
										</SidebarMenuButton>
									</SidebarMenuItem>
								))
							)}
							{activeOrg && (
								<SidebarMenuItem>
									<CreateProjectDialog
										orgId={activeOrg.id}
										trigger={
											<SidebarMenuButton className="text-muted-foreground">
												<Plus />
												<span>New project</span>
											</SidebarMenuButton>
										}
									/>
								</SidebarMenuItem>
							)}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>

				{/* ── Manage (data-driven from URL context) ── */}
				{manageItems.length > 0 && (
					<SidebarGroup>
						<SidebarGroupLabel>Manage</SidebarGroupLabel>
						<SidebarGroupContent>
							<SidebarMenu>
								{manageItems.map((item) => (
									<SidebarMenuItem key={item.href}>
										<SidebarMenuButton
											tooltip={item.label}
											render={(props) => <Link {...props} to={item.href} />}
										>
											<item.icon />
											<span>{item.label}</span>
										</SidebarMenuButton>
									</SidebarMenuItem>
								))}
							</SidebarMenu>
						</SidebarGroupContent>
					</SidebarGroup>
				)}
			</SidebarContent>

			{/* ── Footer: User menu ── */}
			<SidebarFooter>
				<SidebarMenu>
					<SidebarMenuItem>
						<DropdownMenu>
							<DropdownMenuTrigger
								render={
									<SidebarMenuButton size="lg" className="data-open:bg-sidebar-accent">
										<Avatar size="sm" fallback={initials} />
										<div className="grid flex-1 text-left text-sm leading-tight">
											<span className="truncate text-xs font-medium">{me?.email ?? ""}</span>
										</div>
										<ChevronDown className="ml-auto size-3.5" />
									</SidebarMenuButton>
								}
							/>
							<DropdownMenuContent className="min-w-56" align="start" side="top">
								<DropdownMenuItem render={(props) => <Link to="/settings" {...props} />}>
									<Settings />
									Account Settings
								</DropdownMenuItem>
								{activeOrg && (
									<DropdownMenuItem
										render={(props) => (
											<Link
												to="/orgs/$orgId/settings"
												params={{ orgId: activeOrg.id }}
												{...props}
											/>
										)}
									>
										<Building2 />
										Org Settings
									</DropdownMenuItem>
								)}
								<DropdownMenuSeparator />
								<DropdownMenuItem onClick={handleLock}>
									<Lock />
									Lock vault
								</DropdownMenuItem>
								<DropdownMenuItem variant="destructive" onClick={handleSignOut}>
									<LogOut />
									Sign out
								</DropdownMenuItem>
							</DropdownMenuContent>
						</DropdownMenu>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>
		</Sidebar>
	)
}
