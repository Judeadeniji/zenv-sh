import { createFileRoute, Outlet, redirect } from "@tanstack/react-router"
import { projectsQueryOptions } from "#/lib/queries/projects"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId")({
	beforeLoad: async ({ context, params }) => {
		const projects = await context.queryClient.ensureQueryData(projectsQueryOptions(params.orgId))
		const projectList = (projects as { projects?: { id: string; name: string }[] }).projects ?? []
		const project = projectList.find((p) => p.id === params.projectId)

		if (!project) {
			throw redirect({
				to: "/orgs/$orgId",
				params: { orgId: params.orgId },
			})
		}

		return { projectId: params.projectId, project }
	},
	component: () => <Outlet />,
})
