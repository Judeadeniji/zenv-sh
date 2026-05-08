import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Avatar } from "#/components/ui/avatar";
import { Spinner } from "#/components/ui/spinner";
import { Button } from "#/components/ui/button";
import { CreateProjectDialog } from "#/components/create-project-dialog";
import { InviteMemberDialog } from "#/components/invite-member-dialog";
import { orgQueryOptions, orgMembersQueryOptions } from "#/lib/queries/orgs";
import { projectsQueryOptions } from "#/lib/queries/projects";
import { tokensQueryOptions } from "#/lib/queries/tokens";
import { secretsQueryOptions } from "#/lib/queries/secrets";
import { useNavStore } from "#/lib/stores/nav";
import {
  FolderKey,
  Plus,
  UserPlus,
  Settings,
  ChevronRight,
  KeyRound,
  FileKey,
  ArrowUpRight,
} from "lucide-react";

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/")({
  component: OrgDashboard,
});

function OrgDashboard() {
  const { orgId } = Route.useParams();
  const { data: org } = useQuery(orgQueryOptions(orgId));
  const { data: projectsData, isLoading: projectsLoading } = useQuery(
    projectsQueryOptions(orgId),
  );
  const { data: membersData, isLoading: membersLoading } = useQuery(
    orgMembersQueryOptions(orgId),
  );

  const orgName = (org as { name?: string })?.name ?? "Organization";
  const projects = projectsData?.projects ?? [];
  const members = membersData?.members ?? [];

  if (projectsLoading || membersLoading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="w-full">

      {/* ── Page Header ── */}
      <div className="mb-10">
        <div className="mb-3 flex items-center gap-1.5 text-xs text-muted-foreground">
          <span>Dashboard</span>
          <ChevronRight className="size-3" />
          <span className="text-foreground">{orgName}</span>
        </div>

        <div className="flex items-end justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">{orgName}</h1>
            <p className="mt-1 text-xs text-muted-foreground">
              <span className="tabular-nums">{projects.length}</span> project{projects.length !== 1 ? "s" : ""}
              {" · "}
              <span className="tabular-nums">{members.length}</span> member{members.length !== 1 ? "s" : ""}
            </p>
          </div>

          <div className="flex items-center gap-1.5">
            <InviteMemberDialog
              orgId={orgId}
              trigger={
                <Button
                  size="sm"
                  className="h-8 gap-2 border-border text-xs font-normal"
                >
                  <UserPlus className="size-3.5" />
                  Invite member
                </Button>
              }
            />
            <Link to="/orgs/$orgId/settings" params={{ orgId }}>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
              >
                <Settings className="size-4" />
                <span className="sr-only">Settings</span>
              </Button>
            </Link>
          </div>
        </div>
      </div>

      {/* ── Projects Table ── */}
      <section className="mb-10">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
            Projects
          </h2>
          <CreateProjectDialog
            orgId={orgId}
            trigger={
              <button className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground">
                <Plus className="size-3" />
                New project
              </button>
            }
          />
        </div>

        {projects.length === 0 ? (
          <EmptyProjectsState orgId={orgId} />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            {/* Column headers */}
            <div className="grid grid-cols-[1fr_auto_auto_auto] border-b border-border bg-muted/30 px-4 py-2">
              <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                Name
              </span>
              <span className="w-28 text-right text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                Secrets
              </span>
              <span className="w-24 text-right text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                Tokens
              </span>
              <span className="w-8" />
            </div>

            {projects.map((project, i) => (
              <ProjectRow
                key={project.id}
                orgId={orgId}
                project={project}
                isLast={i === projects.length - 1}
              />
            ))}
          </div>
        )}
      </section>

      {/* ── Members + Quick Nav ── */}
      <div className="grid grid-cols-[1fr_160px] gap-8">

        {/* Members */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
              Members
            </h2>
            <Link
              to="/orgs/$orgId/members"
              params={{ orgId }}
              search={{}}
              className="flex items-center gap-0.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              Manage <ArrowUpRight className="size-3" />
            </Link>
          </div>

          <div className="overflow-hidden rounded-lg border border-border">
            {members.slice(0, 6).map((m, i) => (
              <div
                key={m.id}
                className={`flex items-center gap-3 px-3.5 py-2.5 transition-colors hover:bg-muted/40 ${
                  i < Math.min(members.length, 6) - 1 ? "border-b border-border" : ""
                }`}
              >
                <Avatar size="sm" fallback={getInitials(m.name!, m.email)} />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm leading-none">
                    {m.name || m.email || "Unnamed"}
                  </p>
                  {m.name && (
                    <p className="mt-0.5 truncate text-[11px] text-muted-foreground">
                      {m.email}
                    </p>
                  )}
                </div>
                <RoleBadge role={m.role} />
              </div>
            ))}

            {members.length > 6 && (
              <Link
                to="/orgs/$orgId/members"
                params={{ orgId }}
                search={{}}
                className="flex items-center justify-center border-t border-border py-2.5 text-xs text-muted-foreground transition-colors hover:bg-muted/40 hover:text-foreground"
              >
                +{members.length - 6} more
              </Link>
            )}
          </div>
        </section>

        {/* Quick nav rail */}
        <div className="shrink-0">
          <h2 className="mb-3 text-xs font-medium uppercase tracking-widest text-muted-foreground">
            Navigate
          </h2>
          <div className="space-y-px">
            {[
              { label: "All projects", to: "/orgs/$orgId/projects" as const, search: undefined },
              { label: "Members", to: "/orgs/$orgId/members" as const, search: {} },
              { label: "Settings", to: "/orgs/$orgId/settings" as const, search: undefined },
            ].map(({ label, to, search }) => (
              <Link
                key={to}
                to={to}
                params={{ orgId }}
                {...(search !== undefined ? { search } : {})}
                className="flex items-center justify-between rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                {label}
                <ChevronRight className="size-3 opacity-40" />
              </Link>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Sub-components ── */

function ProjectRow({
  orgId,
  project,
  isLast,
}: {
  orgId: string;
  project: { id?: string; name?: string };
  isLast: boolean;
}) {
  const env = useNavStore((s) => s.activeEnvironment);
  const { data: secretsData } = useQuery({
    ...secretsQueryOptions(project.id!, env),
    staleTime: 30_000,
  });
  const { data: tokensData } = useQuery({
    ...tokensQueryOptions(project.id!),
    staleTime: 30_000,
  });

  const secretCount = secretsData?.meta?.total ?? 0;
  const tokenCount = tokensData?.meta?.total ?? 0;

  return (
    <Link
      to="/orgs/$orgId/projects/$projectId"
      params={{ orgId, projectId: project.id! }}
      className={`group grid grid-cols-[1fr_auto_auto_auto] items-center px-4 py-3 transition-colors hover:bg-muted/40 ${
        !isLast ? "border-b border-border" : ""
      }`}
    >
      <div className="flex items-center gap-3">
        <div className="flex size-6 shrink-0 items-center justify-center rounded border border-border bg-background text-muted-foreground transition-colors group-hover:border-foreground/20 group-hover:text-foreground">
          <FolderKey className="size-3" />
        </div>
        <span className="text-sm font-medium">{project.name}</span>
      </div>

      <div className="w-28 text-right">
        <span className="flex items-center justify-end gap-1.5 text-xs tabular-nums text-muted-foreground">
          <KeyRound className="size-3" />
          {secretCount}
        </span>
      </div>

      <div className="w-24 text-right">
        <span className="flex items-center justify-end gap-1.5 text-xs tabular-nums text-muted-foreground">
          <FileKey className="size-3" />
          {tokenCount}
        </span>
      </div>

      <div className="w-8 text-right">
        <ChevronRight className="ml-auto size-3.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
      </div>
    </Link>
  );
}

function EmptyProjectsState({ orgId }: { orgId: string }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16">
      <div className="flex size-10 items-center justify-center rounded-full border border-border bg-muted">
        <FolderKey className="size-4 text-muted-foreground" />
      </div>
      <p className="mt-4 text-sm font-medium">No projects yet</p>
      <p className="mt-1 max-w-[220px] text-center text-xs text-muted-foreground">
        Projects hold your encrypted secrets, scoped by environment.
      </p>
      <CreateProjectDialog
        orgId={orgId}
        trigger={
          <Button size="sm" className="mt-5 h-7 gap-1.5 text-xs">
            <Plus className="size-3.5" />
            Create project
          </Button>
        }
      />
    </div>
  );
}

function RoleBadge({ role }: { role?: string }) {
  if (role === "owner") {
    return (
      <span className="rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
        owner
      </span>
    );
  }
  if (role === "admin") {
    return (
      <span className="rounded-full border border-border bg-muted px-2 py-0.5 text-[10px] font-medium text-foreground">
        admin
      </span>
    );
  }
  return (
    <span className="px-2 py-0.5 text-[10px] text-muted-foreground">
      member
    </span>
  );
}

function getInitials(name?: string, email?: string): string {
  const source = name || email || "?";
  return source
    .split(/[\s@]/)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase() ?? "")
    .join("");
}