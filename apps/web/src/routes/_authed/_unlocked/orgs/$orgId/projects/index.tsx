import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { z } from "zod"
import { type ColumnDef } from "@tanstack/react-table"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "#/components/ui/select"
import { DataTable } from "#/components/data-table"
import { SearchInput } from "#/components/search-input"
import { CreateProjectDialog } from "#/components/create-project-dialog"
import { projectsQueryOptions } from "#/lib/queries/projects"
import { FolderKey, Plus, ArrowRight } from "lucide-react"
import { useMemo } from "react"

const searchSchema = z.object({
  page: z.number().catch(1),
  per_page: z.number().catch(20),
  search: z.string().optional(),
  sort_by: z.string().optional(),
  sort_dir: z.enum(["asc", "desc"]).optional(),
}).partial()

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/")({
  validateSearch: searchSchema,
  component: ProjectsPage,
})

interface ProjectRow {
  id: string
  name: string
  organization_id?: string
  created_at?: string
}

function ProjectsPage() {
  const { orgId } = Route.useParams()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()

  const { data, isLoading } = useQuery(projectsQueryOptions(orgId, search))
  const projects: ProjectRow[] = (data as { projects?: ProjectRow[] })?.projects ?? []

  const columns: ColumnDef<ProjectRow, unknown>[] = useMemo(() => [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-2.5">
          <div className="flex size-7 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <FolderKey className="size-3.5" />
          </div>
          <span className="text-sm font-medium">{row.original.name}</span>
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.created_at ? new Date(row.original.created_at).toLocaleDateString() : "—"}
        </span>
      ),
    },
    {
      id: "open",
      header: "",
      cell: ({ row }) => (
        <div className="text-right">
          <Button
            variant="ghost"
            size="xs"
            render={<Link to="/orgs/$orgId/projects/$projectId" params={{ orgId, projectId: row.original.id }} />}
          >
            Open <ArrowRight className="ml-1 size-3" />
          </Button>
        </div>
      ),
    },
  ], [])

  const sortFilters = useMemo(() => [
    {
      id: "name-asc",
      label: "Name A→Z",
      value: "name-asc",
    },
    {
      id: "name-desc",
      label: "Name Z→A",
      value: "name-desc",
    },
    {
      id: "created_at-desc",
      label: "Newest first",
      value: "created_at-desc",
    },
    {
      id: "created_at-asc",
      label: "Oldest first",
      value: "created_at-asc",
    },
  ], [])

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
        <CreateProjectDialog
          orgId={orgId}
          trigger={<Button type="button" size="sm"><Plus /> New project</Button>}
        />
      </div>

      <div className="mb-4 flex items-center gap-3">
        <SearchInput
          placeholder="Search projects..."
          value={search.search}
          onChange={(val) => {
            navigate({ search: (prev) => ({ ...prev, search: val || undefined, page: 1 }), replace: true })
          }}
        />
        <Select
          value={search.sort_by ? `${search.sort_by}-${search.sort_dir}` : "created_at-desc"}
          onValueChange={(val) => {
            if (!val) return
            const [by, dir] = val.split("-") as [string, "asc" | "desc"]
            navigate({
              search: (prev) => ({
                ...prev,
                sort_by: by,
                sort_dir: dir,
                page: 1,
              }),
              replace: true,
            })
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Sort" />
          </SelectTrigger>
          <SelectContent>
            {sortFilters.map((filter) => (
              <SelectItem key={filter.id} value={filter.value}>
                {filter.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <DataTable
        columns={columns}
        data={projects}
        pagination={data?.meta ? {
          page: data.meta.page ?? 1,
          totalPages: data.meta.total_pages ?? 1,
          total: data.meta.total ?? 0,
          onPageChange: (p) => navigate({ search: (prev) => ({ ...prev, page: p }) })
        } : undefined}
        onRowClick={(row) => {
          navigate({ to: "/orgs/$orgId/projects/$projectId", params: { orgId, projectId: row.original.id } })
        }}
        emptyIcon={<FolderKey />}
        emptyTitle="No projects yet"
        emptyDescription="Projects contain your secrets, tokens, and audit logs. Create one to get started."
        emptyAction={
          <CreateProjectDialog
            orgId={orgId}
            trigger={<Button type="button" size="sm"><Plus /> Create a project</Button>}
          />
        }
      />
    </div>
  )
}

function PageHeader() {
  return (
    <div>
      <h1 className="text-lg font-semibold">Projects</h1>
      <p className="mt-1 text-sm text-muted-foreground">Manage the projects in your organization.</p>
    </div>
  )
}
