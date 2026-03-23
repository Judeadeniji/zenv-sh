import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { encrypt, hashName } from "@zenv/amnesia"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Textarea } from "#/components/ui/textarea"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useProjectDEK } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import { api } from "#/lib/api-client"
import { queryKeys } from "#/lib/keys"
import { toBase64 } from "#/lib/encoding"
import { useQueryClient } from "@tanstack/react-query"
import { AlertCircle, Check } from "lucide-react"

const importSchema = z.object({
	content: z.string().min(1, "Paste content to import"),
})

interface ImportSecretsDialogProps {
	projectId: string
	trigger: React.ReactElement
}

function parseEnv(content: string): { name: string; value: string }[] {
	const entries: { name: string; value: string }[] = []
	for (const line of content.split("\n")) {
		const trimmed = line.trim()
		if (!trimmed || trimmed.startsWith("#")) continue
		const eqIndex = trimmed.indexOf("=")
		if (eqIndex === -1) continue
		const name = trimmed.slice(0, eqIndex).trim()
		let value = trimmed.slice(eqIndex + 1).trim()
		// Strip surrounding quotes
		if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
			value = value.slice(1, -1)
		}
		if (name) entries.push({ name, value })
	}
	return entries
}

export function ImportSecretsDialog({ projectId, trigger }: ImportSecretsDialogProps) {
	const [open, setOpen] = useState(false)
	const [importing, setImporting] = useState(false)
	const [result, setResult] = useState<{ success: number; failed: number } | null>(null)
	const [error, setError] = useState<string | null>(null)
	const environment = useNavStore((s) => s.activeEnvironment)
	const { data: projectDEK } = useProjectDEK(projectId)
	const qc = useQueryClient()

	const form = useForm({
		resolver: zodResolver(importSchema),
		defaultValues: { content: "" },
	})

	const onSubmit = async (data: { content: string }) => {
		if (!projectDEK) return
		setImporting(true)
		setError(null)
		setResult(null)

		const entries = parseEnv(data.content)
		if (entries.length === 0) {
			setError("No valid key=value pairs found")
			setImporting(false)
			return
		}

		let success = 0
		let failed = 0

		for (const entry of entries) {
			try {
				const nameHashBytes = await hashName(entry.name, projectDEK)
				const payload = new TextEncoder().encode(JSON.stringify({ name: entry.name, value: entry.value }))
				const { ciphertext, nonce } = await encrypt(payload, projectDEK)

				const { error: apiErr } = await api().POST("/secrets", {
					body: {
						project_id: projectId,
						environment,
						name_hash: toBase64(nameHashBytes),
						ciphertext: toBase64(ciphertext),
						nonce: toBase64(nonce),
					} as never,
				})

				if (apiErr) failed++
				else success++
			} catch {
				failed++
			}
		}

		setResult({ success, failed })
		setImporting(false)
		qc.invalidateQueries({ queryKey: queryKeys.secrets.list(projectId) })
	}

	const handleClose = (v: boolean) => {
		if (!v) {
			form.reset()
			setResult(null)
			setError(null)
		}
		setOpen(v)
	}

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogTrigger render={trigger} />
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Bulk Import</DialogTitle>
					<DialogDescription>
						Paste key-value pairs to bulk-import secrets. Supports .env format.
					</DialogDescription>
				</DialogHeader>

				{result ? (
					<div className="space-y-3 py-2">
						<div className="flex items-center gap-2 text-sm">
							<Check className="size-4 text-success" />
							<span>{result.success} secret{result.success !== 1 ? "s" : ""} imported</span>
						</div>
						{result.failed > 0 && (
							<p className="text-xs text-destructive">{result.failed} failed (may already exist)</p>
						)}
						<DialogFooter>
							<Button variant="solid" size="sm" onClick={() => handleClose(false)}>Done</Button>
						</DialogFooter>
					</div>
				) : (
					<form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
						{error && (
							<Alert variant="danger">
								<AlertCircle />
								<AlertDescription>{error}</AlertDescription>
							</Alert>
						)}

						<div className="space-y-1.5">
							<Label htmlFor="env-content" className="text-xs">Content</Label>
							<Textarea
								id="env-content"
								placeholder={"api-key=sk_live_...\ndb-password=s3cret\n# Comments are ignored"}
								className="font-mono text-xs"
								rows={8}
								{...form.register("content")}
							/>
							<p className="text-xs text-muted-foreground">
								Comments (#) and empty lines are ignored. Quoted values are unquoted.
							</p>
						</div>

						<DialogFooter>
							<DialogClose>
								<Button variant="ghost" size="sm" type="button">Cancel</Button>
							</DialogClose>
							<Button
								type="submit"
								variant="solid"
								size="sm"
								isLoading={importing}
								disabled={!projectDEK}
							>
								Import
							</Button>
						</DialogFooter>
					</form>
				)}
			</DialogContent>
		</Dialog>
	)
}
