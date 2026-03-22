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
import { useNavStore } from "#/lib/stores/nav"
import { useQuery } from "@tanstack/react-query"
import { orgsQueryOptions } from "#/lib/queries/orgs"
import { projectsQueryOptions } from "#/lib/queries/projects"
import ThemeToggle from "#/components/ThemeToggle"
import { Link } from "@tanstack/react-router"

export function AppHeader() {
	const activeOrgId = useNavStore((s) => s.activeOrgId)
	const activeProjectId = useNavStore((s) => s.activeProjectId)

	const { data: orgsData } = useQuery(orgsQueryOptions)
	const orgList = (orgsData as { organizations?: { id: string; name: string }[] })?.organizations ?? []
	const activeOrg = orgList.find((o) => o.id === activeOrgId)

	const { data: projectsData } = useQuery({
		...projectsQueryOptions(activeOrg?.id ?? ""),
		enabled: !!activeOrg,
	})
	const projectList = (projectsData as { projects?: { id: string; name: string }[] })?.projects ?? []
	const activeProject = projectList.find((p) => p.id === activeProjectId)

	return (
		<header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-4">
			<SidebarTrigger className="-ml-1" />
			<Separator orientation="vertical" className="mr-2 h-4!" />

			<Breadcrumb>
				<BreadcrumbList>
					{activeOrg && (
						<BreadcrumbItem>
							<BreadcrumbLink 
								render={(p) => <Link {...p} to="/" />}
							>{activeOrg.name}</BreadcrumbLink>
						</BreadcrumbItem>
					)}
					{activeProject && (
						<>
							<BreadcrumbSeparator />
							<BreadcrumbItem>
								<BreadcrumbPage>{activeProject.name}</BreadcrumbPage>
							</BreadcrumbItem>
						</>
					)}
				</BreadcrumbList>
			</Breadcrumb>

			<div className="ml-auto">
				<ThemeToggle />
			</div>
		</header>
	)
}
