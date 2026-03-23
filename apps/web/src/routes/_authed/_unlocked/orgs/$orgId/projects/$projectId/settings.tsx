import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Spinner } from "#/components/ui/spinner"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { OneTimeDisplay } from "#/components/ui/one-time-display"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { SettingsRow, SettingsDivider } from "#/components/settings/settings-row"
import { projectQueryOptions, useProjectKey, useDeleteProject } from "#/lib/queries/projects"
import { RotateDEKDialog } from "#/components/rotate-dek-dialog"
import { AlertCircle, Copy, Check, RefreshCw } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/orgs/$orgId/projects/$projectId/settings")({
	component: ProjectSettingsPage,
})

function ProjectSettingsPage() {
	const { orgId, projectId } = Route.useParams()
	const { data: project } = useQuery(projectQueryOptions(projectId))
	const name = (project as { name?: string })?.name ?? projectId
	const createdAt = (project as { created_at?: string })?.created_at

	return (
		<div>
			<div className="mb-2">
				<h1 className="text-xl font-semibold tracking-tight">Project Settings</h1>
				<p className="mt-1 text-sm text-muted-foreground">
					Configuration for {name}.
				</p>
			</div>

			<GeneralSection projectId={projectId} name={name} createdAt={createdAt} />
			<SettingsDivider />
			<ProjectKeyRow projectId={projectId} />
			<SettingsDivider />
			<KeyRotationRow projectId={projectId} />
			<SettingsDivider />
			<DangerZone orgId={orgId} projectId={projectId} name={name} />
		</div>
	)
}

function GeneralSection({ projectId, name, createdAt }: { projectId: string; name: string; createdAt?: string }) {
	const [copied, setCopied] = useState(false)

	const handleCopy = () => {
		navigator.clipboard?.writeText(projectId).then(() => {
			setCopied(true)
			setTimeout(() => setCopied(false), 2000)
		})
	}

	return (
		<SettingsRow title="General" description="Basic project information.">
			<div className="space-y-4">
				<div>
					<label className="text-xs font-medium text-muted-foreground">Project ID</label>
					<div className="mt-1 flex items-center gap-2">
						<code className="flex-1 rounded-md bg-muted px-2.5 py-1.5 font-mono text-xs select-all">{projectId}</code>
						<Button variant="outline" size="icon-sm" onClick={handleCopy}>
							{copied ? <Check className="text-success" /> : <Copy />}
						</Button>
					</div>
				</div>
				<div>
					<label className="text-xs font-medium text-muted-foreground">Name</label>
					<p className="mt-1 text-sm">{name}</p>
				</div>
				{createdAt && (
					<div>
						<label className="text-xs font-medium text-muted-foreground">Created</label>
						<p className="mt-1 text-sm">{new Date(createdAt).toLocaleDateString()}</p>
					</div>
				)}
			</div>
		</SettingsRow>
	)
}

function ProjectKeyRow({ projectId }: { projectId: string }) {
	const [revealed, setRevealed] = useState(false)
	const { data: projectKey, error, isLoading } = useProjectKey(projectId)

	if (!revealed) {
		return (
			<SettingsRow title="Project Key" description="Used by the CLI/SDK to decrypt secrets. Set as ZENV_PROJECT_KEY.">
				<Button variant="outline" size="sm" onClick={() => setRevealed(true)}>
					Reveal
				</Button>
			</SettingsRow>
		)
	}

	return (
		<SettingsRow title="Project Key" description="Used by the CLI/SDK to decrypt secrets. Set as ZENV_PROJECT_KEY.">
			{isLoading && <Spinner />}
			{error && (
				<Alert variant="danger">
					<AlertCircle />
					<AlertDescription>{error.message}</AlertDescription>
				</Alert>
			)}
			{projectKey && (
				<div>
					<OneTimeDisplay value={projectKey} label="ZENV_PROJECT_KEY" masked={false} />
					<p className="mt-2 text-[11px] text-muted-foreground">
						Unwrapped in your browser using your private key. The server never sees this value.
					</p>
				</div>
			)}
		</SettingsRow>
	)
}

function KeyRotationRow({ projectId }: { projectId: string }) {
	return (
		<SettingsRow
			title="Key Rotation"
			description="Re-encrypt all secrets with a fresh DEK. Use this if you suspect key compromise or as a routine security measure."
		>
			<RotateDEKDialog
				projectId={projectId}
				trigger={
					<Button variant="outline" size="sm">
						<RefreshCw className="size-3.5" />
						Rotate DEK
					</Button>
				}
			/>
		</SettingsRow>
	)
}

function DangerZone({ orgId, projectId, name }: { orgId: string; projectId: string; name: string }) {
	const [confirmOpen, setConfirmOpen] = useState(false)
	const [confirmText, setConfirmText] = useState("")
	const navigate = useNavigate()
	const deleteProject = useDeleteProject()

	const handleDelete = () => {
		deleteProject.mutate(
			{ projectId },
			{
				onSuccess: () => {
					setConfirmOpen(false)
					navigate({ to: "/orgs/$orgId", params: { orgId } })
				},
			},
		)
	}

	return (
		<SettingsRow
			title="Danger zone"
			description="Irreversible actions. Proceed with caution."
			className="border-t-destructive/30"
		>
			<div className="rounded-lg border border-destructive/30 p-4">
				<div className="flex items-center justify-between">
					<div>
						<p className="text-sm font-medium">Delete this project</p>
						<p className="mt-0.5 text-xs text-muted-foreground">
							All secrets, tokens, and key grants will be permanently deleted.
						</p>
					</div>
					<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
						<DialogTrigger render={<Button variant="danger" size="sm">Delete project</Button>} />
						<DialogContent>
							<DialogHeader>
								<DialogTitle>Delete {name}?</DialogTitle>
								<DialogDescription>
									This action cannot be undone. All secrets, service tokens, and key grants
									in this project will be permanently deleted.
								</DialogDescription>
							</DialogHeader>

							<div className="py-2">
								<label className="text-xs font-medium text-muted-foreground">
									Type <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-semibold">{name}</code> to confirm
								</label>
								<Input
									className="mt-1.5"
									placeholder={name}
									value={confirmText}
									onChange={(e) => setConfirmText(e.target.value)}
									autoFocus
								/>
							</div>

							<DialogFooter>
								<DialogClose>
									<Button variant="ghost" size="sm" type="button">Cancel</Button>
								</DialogClose>
								<Button
									variant="danger"
									size="sm"
									disabled={confirmText !== name}
									isLoading={deleteProject.isPending}
									onClick={handleDelete}
								>
									Delete permanently
								</Button>
							</DialogFooter>

							{deleteProject.error && (
								<Alert variant="danger" className="mt-2">
									<AlertCircle />
									<AlertDescription>{deleteProject.error.message}</AlertDescription>
								</Alert>
							)}
						</DialogContent>
					</Dialog>
				</div>
			</div>
		</SettingsRow>
	)
}
