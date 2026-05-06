import { useState, useCallback } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
	Dialog,
	DialogTrigger,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
	DialogFooter,
	DialogClose,
} from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Progress, ProgressLabel, ProgressValue } from "#/components/ui/progress"
import { secretsQueryOptions } from "#/lib/queries/secrets"
import { useKeyGrantMembers } from "#/lib/queries/rotation"
import { useProjectDEK } from "#/lib/queries/projects"
import { queryKeys } from "#/lib/keys"
import { rotateProjectDEK, type RotationProgress } from "#/lib/rotation"
import { AlertCircle, CheckCircle, RefreshCw, Shield } from "lucide-react"

interface RotateDEKDialogProps {
	projectId: string
	trigger: React.ReactElement
}

type DialogStep = "confirm" | "progress" | "complete" | "error"

export function RotateDEKDialog({ projectId, trigger }: RotateDEKDialogProps) {
	const [open, setOpen] = useState(false)
	const [step, setStep] = useState<DialogStep>("confirm")
	const [progress, setProgress] = useState<RotationProgress>({
		phase: "preparing",
		staged: 0,
		total: 0,
	})
	const [errorMessage, setErrorMessage] = useState<string | null>(null)

	const qc = useQueryClient()
	const { data: projectDEK } = useProjectDEK(projectId)
	const { data: members } = useKeyGrantMembers(projectId)

	// Count secrets across all environments
	const { data: devData } = useQuery({ ...secretsQueryOptions(projectId, "development"), staleTime: 30_000 })
	const { data: stgData } = useQuery({ ...secretsQueryOptions(projectId, "staging"), staleTime: 30_000 })
	const { data: prdData } = useQuery({ ...secretsQueryOptions(projectId, "production"), staleTime: 30_000 })

	const devCount = ((devData as { secrets?: unknown[] })?.secrets ?? []).length
	const stgCount = ((stgData as { secrets?: unknown[] })?.secrets ?? []).length
	const prdCount = ((prdData as { secrets?: unknown[] })?.secrets ?? []).length
	const totalSecrets = devCount + stgCount + prdCount

	const handleProgress = useCallback((p: RotationProgress) => {
		setProgress(p)
		if (p.phase === "complete") setStep("complete")
		if (p.phase === "error") {
			setErrorMessage(p.error ?? "Rotation failed")
			setStep("error")
		}
	}, [])

	const handleStart = async () => {
		if (!projectDEK || !members) return
		setStep("progress")
		setErrorMessage(null)

		try {
			await rotateProjectDEK({
				projectId,
				oldProjectDEK: projectDEK,
				members: members as { user_id: string; public_key: string }[],
				onProgress: handleProgress,
			})
			// Invalidate all caches after successful rotation
			await qc.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) })
			await qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
		} catch {
			// Error already handled via onProgress
		}
	}

	const handleClose = (v: boolean) => {
		// Prevent closing during active rotation
		if (!v && step === "progress") return
		setOpen(v)
		if (!v) {
			setStep("confirm")
			setProgress({ phase: "preparing", staged: 0, total: 0 })
			setErrorMessage(null)
		}
	}

	const pct = progress.total > 0 ? Math.round((progress.staged / progress.total) * 100) : 0

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogTrigger render={trigger} nativeButton={false} />
			<DialogContent>
				{step === "confirm" && (
					<>
						<DialogHeader>
							<DialogTitle className="flex items-center gap-2">
								<RefreshCw className="size-4" />
								Rotate Encryption Keys
							</DialogTitle>
							<DialogDescription>
								Re-encrypt all secrets with a fresh Data Encryption Key. Use this if you suspect
								key compromise or as a routine security measure.
							</DialogDescription>
						</DialogHeader>

						<div className="space-y-3 py-2">
							<div className="rounded-md bg-muted p-3 text-xs space-y-1.5">
								<div className="flex justify-between">
									<span className="text-muted-foreground">Secrets to re-encrypt</span>
									<span className="font-medium tabular-nums">{totalSecrets}</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Team members</span>
									<span className="font-medium tabular-nums">{members?.length ?? 0}</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Environments</span>
									<span className="font-medium">
										{[devCount > 0 && "dev", stgCount > 0 && "staging", prdCount > 0 && "prod"].filter(Boolean).join(", ") || "none"}
									</span>
								</div>
							</div>

							<Alert variant="warning">
								<Shield />
								<AlertDescription className="text-xs">
									All decryption and re-encryption happens in your browser. The server never sees plaintext secrets.
								</AlertDescription>
							</Alert>
						</div>

						<DialogFooter>
							<DialogClose>
								<Button variant="ghost" size="sm" type="button">Cancel</Button>
							</DialogClose>
							<Button
								variant="solid"
								size="sm"
								onClick={handleStart}
								disabled={!projectDEK || !members || totalSecrets === 0}
							>
								<RefreshCw className="size-3.5" />
								Rotate Keys
							</Button>
						</DialogFooter>
					</>
				)}

				{step === "progress" && (
					<>
						<DialogHeader>
							<DialogTitle>
								{progress.phase === "preparing" && "Preparing..."}
								{progress.phase === "staging" && "Re-encrypting secrets..."}
								{progress.phase === "committing" && "Committing..."}
							</DialogTitle>
							<DialogDescription>
								Do not close this window. All crypto operations run in your browser.
							</DialogDescription>
						</DialogHeader>

						<div className="space-y-4 py-4">
							<Progress value={pct}>
								<ProgressLabel>{progress.phase === "committing" ? "Finalizing" : "Progress"}</ProgressLabel>
								<ProgressValue />
							</Progress>

							<p className="text-center text-xs text-muted-foreground tabular-nums">
								{progress.staged} / {progress.total} secrets re-encrypted
							</p>
						</div>
					</>
				)}

				{step === "complete" && (
					<>
						<DialogHeader>
							<DialogTitle className="flex items-center gap-2">
								<CheckCircle className="size-4 text-success" />
								Rotation Complete
							</DialogTitle>
							<DialogDescription>
								All {progress.total} secrets have been re-encrypted with a fresh DEK. Key grants
								updated for {members?.length ?? 0} team member{(members?.length ?? 0) !== 1 ? "s" : ""}.
							</DialogDescription>
						</DialogHeader>

						<DialogFooter>
							<Button variant="solid" size="sm" onClick={() => handleClose(false)}>
								Done
							</Button>
						</DialogFooter>
					</>
				)}

				{step === "error" && (
					<>
						<DialogHeader>
							<DialogTitle>Rotation Failed</DialogTitle>
							<DialogDescription>
								The rotation has been cancelled and cleaned up. Your existing keys are still active.
							</DialogDescription>
						</DialogHeader>

						<Alert variant="danger" className="my-2">
							<AlertCircle />
							<AlertDescription>{errorMessage}</AlertDescription>
						</Alert>

						<DialogFooter>
							<Button variant="ghost" size="sm" onClick={() => handleClose(false)}>
								Close
							</Button>
							<Button variant="solid" size="sm" onClick={handleStart}>
								Retry
							</Button>
						</DialogFooter>
					</>
				)}
			</DialogContent>
		</Dialog>
	)
}
