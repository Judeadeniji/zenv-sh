import { useState, type ReactNode } from "react"
import type { DecryptedSecretRow } from "#/lib/queries/secrets"
import { Button, buttonVariants } from "#/components/ui/button"
import { SecretMimeIcon } from "#/components/secret-mime-icon"
import {
	decodedUtf8Preview,
	downloadFilename,
	formatSecretByteSize,
	isPreviewableMime,
	resolveSecretMime,
} from "#/lib/secret-mime"
import { cn } from "#/lib/utils"
import { Eye, EyeOff, Copy, Check } from "lucide-react"

const TEXT_DISPLAY_MAX = 12_000

type Props = {
	secret: DecryptedSecretRow
	/** Object URL for binary payloads; revoked by parent when sheet closes. */
	blobUrl: string | null
	copied: string | null
	onCopy: (text: string, key: string) => void
	/** When true, text values start visible (e.g. detail sheet). */
	defaultTextRevealed?: boolean
}

export function SecretContentPanel({
	secret,
	blobUrl,
	copied,
	onCopy,
	defaultTextRevealed = true,
}: Props) {
	const [textRevealed, setTextRevealed] = useState(defaultTextRevealed)
	const mime = resolveSecretMime(secret)

	if (secret.kind === "text") {
		const display =
			secret.value.length > TEXT_DISPLAY_MAX
				? `${secret.value.slice(0, TEXT_DISPLAY_MAX)}…`
				: secret.value
		return (
			<div className="space-y-2">
				<div className="flex flex-wrap items-center gap-2">
					<SecretMimeIcon
						name={secret.name}
						value={secret.value}
						kind={secret.kind}
						metadata={secret.metadata}
						resolvedMime={secret.resolvedMime}
						size={18}
						className="shrink-0"
					/>
					<span className="font-mono text-[11px] text-muted-foreground">{mime}</span>
				</div>
				<div className="flex items-start gap-2">
					{textRevealed ? (
						<code className="flex-1 max-h-80 overflow-auto whitespace-pre-wrap wrap-break-word rounded bg-muted px-2 py-1 font-mono text-xs">
							{display}
						</code>
					) : (
						<span className="flex-1 break-all rounded bg-muted px-2 py-1 font-mono text-xs text-muted-foreground">
							{"•".repeat(Math.min(secret.value.length, 64))}
						</span>
					)}
					<div className="flex shrink-0 flex-col gap-0.5">
						<Button
							variant="ghost"
							size="icon-sm"
							type="button"
							onClick={() => setTextRevealed((r) => !r)}
							aria-label={textRevealed ? "Hide value" : "Show value"}
						>
							{textRevealed ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
						</Button>
						<Button variant="ghost" size="icon-sm" type="button" onClick={() => onCopy(secret.value, "value")}>
							{copied === "value" ? <Check className="size-3 text-success" /> : <Copy className="size-3" />}
						</Button>
					</div>
				</div>
			</div>
		)
	}

	const bytes = secret.binary?.byteLength ?? 0
	const preview = isPreviewableMime(mime)
	const fileName = downloadFilename(secret.name, mime, secret.suggestedDownloadFilename)

	let previewBody: ReactNode = null
	if (preview && blobUrl && secret.binary) {
		if (mime.startsWith("image/")) {
			previewBody = (
				<img src={blobUrl} alt="" className="max-h-48 max-w-full rounded-md border object-contain" />
			)
		} else if (mime === "application/pdf") {
			previewBody = (
				<iframe title="PDF preview" src={blobUrl} className="h-64 w-full rounded-md border bg-muted" />
			)
		} else if (mime.startsWith("video/")) {
			previewBody = (
				// biome-ignore lint/a11y/useMediaCaption: arbitrary secret file; no caption track available
				<video src={blobUrl} controls className="max-h-48 max-w-full rounded-md border" />
			)
		} else if (mime.startsWith("audio/")) {
			previewBody = (
				// biome-ignore lint/a11y/useMediaCaption: arbitrary secret file; no caption track available
				<audio src={blobUrl} controls className="w-full" />
			)
		} else if (mime.startsWith("text/") || mime === "application/json" || mime.endsWith("+json")) {
			const raw = decodedUtf8Preview(secret.binary)
			let body = raw
			if (mime === "application/json" || mime.endsWith("+json")) {
				try {
					body = JSON.stringify(JSON.parse(raw) as unknown, null, 2)
				} catch {
					/* keep raw */
				}
			}
			previewBody = (
				<pre className="max-h-48 overflow-auto rounded-md border bg-muted p-2 font-mono text-[11px] whitespace-pre-wrap wrap-break-word">
					{body}
				</pre>
			)
		}
	}

	return (
		<div className="space-y-3">
			<div className="flex flex-wrap items-center gap-2">
				<SecretMimeIcon
					name={secret.name}
					value={secret.value}
					kind={secret.kind}
					metadata={secret.metadata}
					resolvedMime={secret.resolvedMime}
					size={18}
					className="shrink-0"
				/>
				<div className="min-w-0 flex-1">
					<p className="font-mono text-[11px] text-muted-foreground">{mime}</p>
					<p className="text-xs text-muted-foreground">{formatSecretByteSize(bytes)}</p>
				</div>
			</div>
			{blobUrl ? (
				<div className="flex flex-wrap gap-2">
					<a
						href={blobUrl}
						download={fileName}
						className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
					>
						Download
					</a>
					<Button variant="ghost" size="sm" type="button" onClick={() => onCopy(secret.value, "value")}>
						{copied === "value" ? <Check className="size-3.5 text-success" /> : <Copy className="size-3.5" />}
						Copy base64
					</Button>
				</div>
			) : null}
			{previewBody}
			{!preview && <p className="text-xs text-muted-foreground">Preview is not available for this type. Download the file to open it.</p>}
		</div>
	)
}
