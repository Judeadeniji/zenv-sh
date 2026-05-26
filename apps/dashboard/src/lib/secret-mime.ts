import { inferBinarySecretMimeServerFn, inferFileMimeServerFn } from "#/server/mime-lookup-fns"

export type SecretPayloadKind = "text" | "binary"

export type SecretMimeRow = {
	name: string
	value: string
	kind: SecretPayloadKind
	metadata?: { mime_type?: string }
}

export function inferTextMime(value: string): string {
	const t = value.trim()
	if (!t) return "text/plain"
	try {
		JSON.parse(t) as unknown
		return "application/json"
	} catch {
		return "text/plain"
	}
}

/** File upload: browser `type` + extension lookup on the server (TanStack Start). */
export function inferMimeForCreateFile(file: File): Promise<string> {
	return inferFileMimeServerFn({
		data: { filename: file.name, typeHint: file.type || undefined },
	})
}

export async function inferMimeForUpdateClient(
	secret: SecretMimeRow,
	newValue: string,
): Promise<string> {
	if (secret.kind === "binary") {
		return inferBinarySecretMimeServerFn({
			data: { name: secret.name, metadataMime: secret.metadata?.mime_type },
		})
	}
	return inferTextMime(newValue)
}

/**
 * Resolved MIME for UI. Prefer `resolvedMime` from the server batch in `secretPayloadQueryOptions`.
 */
export function resolveSecretMime(row: SecretMimeRow & { resolvedMime?: string }): string {
	if (row.resolvedMime) return row.resolvedMime
	if (row.kind === "text") return inferTextMime(row.value)
	return "application/octet-stream"
}

/** Map to a concrete @untitledui/file-icons type when possible. */
export function mimeToFileIconType(mime: string): string {
	const base = mime.split(";")[0].trim().toLowerCase()
	if (base === "application/pdf") return "pdf"
	if (base.startsWith("image/")) {
		if (base === "image/png") return "png"
		if (base === "image/jpeg" || base === "image/jpg") return "jpg"
		if (base === "image/gif") return "gif"
		if (base === "image/svg+xml") return "svg"
		if (base === "image/webp") return "webp"
		return "image"
	}
	if (base.startsWith("video/")) {
		if (base === "video/mp4") return "mp4"
		return "video"
	}
	if (base.startsWith("audio/")) {
		if (base === "audio/mpeg" || base === "audio/mp3") return "mp3"
		return "audio"
	}
	if (base === "application/json" || base.endsWith("+json")) return "json"
	if (base === "text/html" || base === "application/xhtml+xml") return "html"
	if (base === "application/xml" || base === "text/xml") return "xml"
	if (base === "text/css") return "css"
	if (base === "text/csv") return "csv"
	if (
		base.startsWith("text/") ||
		base === "application/javascript" ||
		base === "application/typescript"
	)
		return "txt"
	if (
		base === "application/vnd.ms-excel" ||
		base === "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	)
		return "xlsx"
	if (base === "application/zip" || base === "application/x-zip-compressed") return "zip"
	return "document"
}

export function isPreviewableMime(mime: string): boolean {
	const base = mime.split(";")[0].trim().toLowerCase()
	if (base === "text/html" || base === "application/xhtml+xml") return false
	if (base.startsWith("image/")) return true
	if (base.startsWith("text/")) return true
	if (base === "application/json" || base.endsWith("+json")) return true
	if (base === "application/pdf") return true
	if (base.startsWith("video/")) return true
	if (base.startsWith("audio/")) return true
	return false
}

/** Download filename: prefer server `suggestedDownloadFilename` from batch resolution. */
export function downloadFilename(
	name: string,
	_mime: string,
	suggestedDownloadFilename?: string | null,
): string {
	if (suggestedDownloadFilename) return suggestedDownloadFilename
	const clean = name.trim() || "download"
	if (clean.includes(".") && !clean.endsWith(".")) {
		const last = clean.lastIndexOf(".")
		if (last > 0 && last < clean.length - 1) return clean
	}
	return clean
}

export const TEXT_PREVIEW_BYTE_CAP = 65_536

export function decodedUtf8Preview(bytes: Uint8Array, maxBytes = TEXT_PREVIEW_BYTE_CAP): string {
	const slice = bytes.byteLength > maxBytes ? bytes.slice(0, maxBytes) : bytes
	return new TextDecoder("utf-8", { fatal: false }).decode(slice)
}

export function formatSecretByteSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
