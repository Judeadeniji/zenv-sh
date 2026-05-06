import { useState } from "react"
import { Link, useLocation, useNavigate, useParams } from "@tanstack/react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { projectsQueryOptions } from "#/lib/queries/projects"
import { useUpdatePreferences } from "#/lib/queries/preferences"
import { meQueryOptions } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"
import { useNavStore } from "#/lib/stores/nav"
import { authClient } from "#/lib/auth-client"
import { api } from "#/lib/api-client"
import { getProjectItems, getOrgItems, getSettingsItems } from "#/lib/nav-items"
import { CreateProjectDialog } from "#/components/create-project-dialog"
import { queryKeys } from "#/lib/keys"
import { Badge } from "#/components/ui/badge"
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupLabel,
	SidebarGroupContent,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuAction,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarMenuSkeleton,
	useSidebar,
} from "#/components/ui/sidebar"
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "#/components/ui/collapsible"
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
	ChevronRight,
	Building2,
	ChevronsUpDown,
	Pin,
	PinOff,
	Users,
	UserStarIcon,
} from "lucide-react"

export function AppSidebar() {
	const location = useLocation()
	const navigate = useNavigate()
	const qc = useQueryClient()
	const { state } = useSidebar()
	const crypto = useAuthStore((s) => s.crypto)
	const { data: me } = useQuery(meQueryOptions)
	const { data: orgsData, isLoading: orgsLoading } = useQuery({
		...orgsQueryOptions(),
		enabled: !!crypto,
	})

	const params = useParams({ strict: false }) as { orgId?: string; projectId?: string }
	const orgId = params.orgId
	const projectId = params.projectId

	const orgList = (orgsData as { organizations?: { id: string; name: string }[] })?.organizations ?? []
	const activeOrg = orgList.find((o) => o.id === orgId) ?? orgList[0]

	const { data: projectsData, isLoading: projectsLoading } = useQuery({
		...projectsQueryOptions(activeOrg?.id ?? ""),
		enabled: !!activeOrg && !!crypto,
	})
	const projectList = (projectsData as { projects?: { id: string; name: string }[] })?.projects ?? []

	const pinnedIds = useNavStore((s) => s.pinnedProjects)
	const pinProject = useNavStore((s) => s.pinProject)
	const unpinProject = useNavStore((s) => s.unpinProject)
	const updatePrefs = useUpdatePreferences()

	const pinned = pinnedIds
		.map((id) => projectList.find((p) => p.id === id))
		.filter(Boolean) as { id: string; name: string }[]
	const unpinned = projectList.filter((p) => !pinnedIds.includes(p.id))
	const hasPins = pinned.length > 0

	const handlePin = (id: string) => {
		pinProject(id)
		const next = [id, ...pinnedIds.filter((p) => p !== id)]
		updatePrefs.mutate({ pinned_projects: next })
	}
	const handleUnpin = (id: string) => {
		unpinProject(id)
		const next = pinnedIds.filter((p) => p !== id)
		updatePrefs.mutate({ pinned_projects: next })
	}

	const initials = me?.email?.slice(0, 2).toUpperCase() ?? "?"

	const projectItems = activeOrg && projectId ? getProjectItems(activeOrg.id, projectId) : []
	const orgItems = activeOrg ? getOrgItems(activeOrg.id, projectId) : []
	const settingsItems = activeOrg ? getSettingsItems(activeOrg.id, projectId) : []

	const { data: incomingRequests } = useQuery({
		queryKey: queryKeys.recovery.incomingRequests,
		queryFn: async () => {
			const { data, error } = await api().GET("/auth/recovery/incoming-requests")
			if (error) return []
			return (data ?? [])
		},
		enabled: !!crypto,
		refetchInterval: 30_000,
	})
	const pendingIncomingCount = (incomingRequests ?? []).filter((r) => r.status === "pending").length

	const handleLock = async () => {
		await api().POST("/auth/lock", {})
		useAuthStore.getState().lock()
		navigate({
			to: "/unlock", search: {
				redirect: location.pathname
			}
		})
	}

	const handleSignOut = async () => {
		await api().POST("/auth/lock", {})
		useAuthStore.getState().lock()
		await authClient.signOut()
		await qc.cancelQueries()
		qc.clear()
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
								<>
									{/* Pinned — always visible */}
									{pinned.map((project) => (
										<ProjectItem
											key={project.id}
											project={project}
											orgId={activeOrg?.id ?? ""}
											isActive={project.id === projectId}
											isPinned
											onTogglePin={() => handleUnpin(project.id)}
										/>
									))}

									{/* Unpinned — collapsible when pins exist */}
									{unpinned.length > 0 && hasPins ? (
										<UnpinnedSection
											projects={unpinned}
											orgId={activeOrg?.id ?? ""}
											activeProjectId={projectId}
											onPin={handlePin}
										/>
									) : (
										unpinned.map((project) => (
											<ProjectItem
												key={project.id}
												project={project}
												orgId={activeOrg?.id ?? ""}
												isActive={project.id === projectId}
												isPinned={false}
												onTogglePin={() => handlePin(project.id)}
											/>
										))
									)}
								</>
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

				{/* ── Project nav ── */}
				{projectItems.length > 0 && (
					<SidebarGroup>
						<SidebarGroupLabel>Project</SidebarGroupLabel>
						<SidebarGroupContent>
							<SidebarMenu>
								{projectItems.map((item) => (
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

				{/* ── Organization nav ── */}
				{orgItems.length > 0 && (
					<SidebarGroup>
						<SidebarGroupLabel>Organization</SidebarGroupLabel>
						<SidebarGroupContent>
							<SidebarMenu>
								{orgItems.map((item) => (
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
								{settingsItems.map((item) => (
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
								<SidebarMenuItem>
								<SidebarMenuButton
									tooltip="Recovery Requests"
									render={(props) => <Link {...props} to="/recovery-requests" />}
								>
									<UserStarIcon />
									<span className="flex-1">Recovery Requests</span>
									{pendingIncomingCount > 0 && state !== "collapsed" && (
										<Badge variant="warning" className="ml-auto">
											{pendingIncomingCount}
										</Badge>
									)}
								</SidebarMenuButton>
							</SidebarMenuItem>
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

/* ── Sub-components ── */

function ProjectItem({
	project,
	orgId,
	isActive,
	isPinned,
	onTogglePin,
}: {
	project: { id: string; name: string }
	orgId: string
	isActive: boolean
	isPinned: boolean
	onTogglePin: () => void
}) {
	return (
		<SidebarMenuItem>
			<SidebarMenuButton
				isActive={isActive}
				tooltip={project.name}
				render={
					<Link
						to="/orgs/$orgId/projects/$projectId"
						params={{ orgId, projectId: project.id }}
					/>
				}
			>
				<FolderKey />
				<span>{project.name}</span>
			</SidebarMenuButton>
			<SidebarMenuAction
				showOnHover
				onClick={(e) => {
					e.preventDefault()
					e.stopPropagation()
					onTogglePin()
				}}
				className={isPinned ? "text-muted-foreground" : "text-muted-foreground/60"}
			>
				{isPinned ? <PinOff /> : <Pin />}
			</SidebarMenuAction>
		</SidebarMenuItem>
	)
}

function UnpinnedSection({
	projects,
	orgId,
	activeProjectId,
	onPin,
}: {
	projects: { id: string; name: string }[]
	orgId: string
	activeProjectId?: string
	onPin: (id: string) => void
}) {
	const [open, setOpen] = useState(false)

	return (
		<Collapsible open={open} onOpenChange={setOpen}>
			<SidebarMenuItem>
				<CollapsibleTrigger
					render={
						<SidebarMenuButton className="text-muted-foreground text-xs">
							<ChevronRight className={`size-3 transition-transform ${open ? "rotate-90" : ""}`} />
							<span>{open ? "Other projects" : `${projects.length} more`}</span>
						</SidebarMenuButton>
					}
				/>
			</SidebarMenuItem>
			<CollapsibleContent>
				{projects.map((project) => (
					<ProjectItem
						key={project.id}
						project={project}
						orgId={orgId}
						isActive={project.id === activeProjectId}
						isPinned={false}
						onTogglePin={() => onPin(project.id)}
					/>
				))}
			</CollapsibleContent>
		</Collapsible>
	)
}
