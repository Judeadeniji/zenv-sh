import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Button } from "#/components/ui/button";
import { Badge } from "#/components/ui/badge";
import { Avatar } from "#/components/ui/avatar";
import { Spinner } from "#/components/ui/spinner";
import { ActionCard } from "#/components/ui/card";
import { CreateProjectDialog } from "#/components/create-project-dialog";
import { InviteMemberDialog } from "#/components/invite-member-dialog";
import { orgQueryOptions, orgMembersQueryOptions } from "#/lib/queries/orgs";
import { projectsQueryOptions } from "#/lib/queries/projects";
import { tokensQueryOptions } from "#/lib/queries/tokens";
import { secretsQueryOptions } from "#/lib/queries/secrets";
import { useNavStore } from "#/lib/stores/nav";
import {
  FolderKey,
  Users,
  Plus,
  UserPlus,
  Settings,
  ArrowRight,
  KeyRound,
  FileKey,
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
      <div className="flex items-center justify-center py-20">
        <Spinner />
      </div>
    );
  }

  return (
    <div>
      {/* ── Header ── */}
      <div className="mb-6">
        <h1 className="text-lg font-semibold">{orgName}</h1>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Organization overview
        </p>
      </div>

      {/* ── Quick Actions ── */}
      <div className="mb-6 grid gap-3 sm:grid-cols-3">
        <CreateProjectDialog
          orgId={orgId}
          trigger={
            <ActionCard className="cursor-pointer transition-all hover:scale-[1.02] shadow-none border">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
                  <Plus className="size-4" />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium">New project</p>
                  <p className="text-[11px] text-muted-foreground">
                    Add a secret vault
                  </p>
                </div>
              </div>
            </ActionCard>
          }
        />
        <InviteMemberDialog
          orgId={orgId}
          trigger={
            <ActionCard className="cursor-pointer transition-all hover:scale-[1.02] shadow-none border">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-blue-500">
                  <UserPlus className="size-4" />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium">Invite member</p>
                  <p className="text-[11px] text-muted-foreground">
                    Grow your team
                  </p>
                </div>
              </div>
            </ActionCard>
          }
        />
        <Link to="/orgs/$orgId/settings" params={{ orgId }}>
          <ActionCard className="h-full cursor-pointer transition-all hover:scale-[1.02] shadow-none border">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <Settings className="size-4" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium">Settings</p>
                <p className="text-[11px] text-muted-foreground">
                  Manage your org
                </p>
              </div>
            </div>
          </ActionCard>
        </Link>
      </div>

      {/* ── Stat Cards ── */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2">
        <Link
          to="/orgs/$orgId/projects"
          params={{ orgId }}
          className="rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
        >
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <FolderKey className="size-4" />
            </div>
            <div>
              <p className="text-2xl font-semibold tabular-nums">
                {projects.length}
              </p>
              <p className="text-xs text-muted-foreground">Projects</p>
            </div>
          </div>
        </Link>
        <Link
          to="/orgs/$orgId/members"
          params={{ orgId }}
          search={{}}
          className="group rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
        >
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-foreground">
              <Users className="size-4" />
            </div>
            <div>
              <p className="text-2xl font-semibold tabular-nums">
                {members.length}
              </p>
              <p className="text-xs text-muted-foreground">Members</p>
            </div>
          </div>
        </Link>
      </div>

      {/* ── Projects List ── */}
      {projects.length === 0 ? (
        <EmptyState orgId={orgId} />
      ) : (
        <section className="mb-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium">Projects</h2>
          </div>
          <div className="divide-y divide-border rounded-lg border border-border">
            {projects.map((project) => (
              <ProjectRow key={project.id} orgId={orgId} project={project} />
            ))}
          </div>
          <div className="mt-3">
            <CreateProjectDialog
              orgId={orgId}
              trigger={
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-muted-foreground"
                >
                  <Plus /> Add project
                </Button>
              }
            />
          </div>
        </section>
      )}

      {/* ── Team Preview ── */}
      <section className="rounded-lg border border-border p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium">Team</h2>
          <Link
            to="/orgs/$orgId/members"
            params={{ orgId }}
            search={{}}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            View all <ArrowRight className="ml-0.5 inline size-3" />
          </Link>
        </div>
        <div className="space-y-2">
          {members.slice(0, 5).map((m) => (
            <div key={m.id} className="flex items-center gap-3">
              <Avatar size="sm" fallback={getInitials(m.name!, m.email)} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {m.name! || m.email || "Unnamed"}
                </p>
              </div>
              <Badge
                variant={
                  m.role === "owner"
                    ? "primary"
                    : m.role === "admin"
                      ? "primary"
                      : "neutral"
                }
                className="text-[10px]"
              >
                {m.role}
              </Badge>
            </div>
          ))}
          {members.length > 5 && (
            <p className="text-xs text-muted-foreground">
              +{members.length - 5} more
            </p>
          )}
        </div>
      </section>
    </div>
  );
}

/* ── Sub-components ── */

function ProjectRow({
  orgId,
  project,
}: {
  orgId: string;
  project: { id?: string; name?: string };
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

  return (
    <Link
      to="/orgs/$orgId/projects/$projectId"
      params={{ orgId, projectId: project.id! }}
      className="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-muted/50"
    >
      <div className="flex size-8 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <FolderKey className="size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{project.name}</p>
      </div>
      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <KeyRound className="size-3" /> {secretsData?.meta?.total || 0}
        </span>
        <span className="flex items-center gap-1">
          <FileKey className="size-3" /> {tokensData?.meta?.total || 0}
        </span>
      </div>
      <ArrowRight className="size-3.5 text-muted-foreground" />
    </Link>
  );
}

function EmptyState({ orgId }: { orgId: string }) {
  return (
    <div className="mb-6 flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-12 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <FolderKey className="size-5" />
      </div>
      <h3 className="mt-4 text-sm font-medium">No projects yet</h3>
      <p className="mt-1 max-w-xs text-xs text-muted-foreground">
        Projects hold your encrypted secrets, organized by environment. Create
        one to get started.
      </p>
      <div className="mt-4">
        <CreateProjectDialog
          orgId={orgId}
          trigger={
            <Button type="button" size="sm">
              <Plus /> Create your first project
            </Button>
          }
        />
      </div>
    </div>
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
