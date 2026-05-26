import mime from "mime-types"

export const OCTET = "application/octet-stream"

/** Normalize a MIME hint or extension using `mime-types` (server-only). */
export function normalizeMimeServer(raw: string): string {
	const trimmed = raw.trim()
	if (!trimmed) return OCTET
	let ct = mime.contentType(trimmed)
	if (!ct) {
		const looked = mime.lookup(trimmed)
		if (looked) ct = mime.contentType(looked)
	}
	if (typeof ct === "string") return ct.split(";")[0].trim().toLowerCase()
	const looked = mime.lookup(trimmed)
	if (looked) return looked.toLowerCase()
	return trimmed.toLowerCase()
}

/**
 * Server-side MIME resolution (no secret `value` — never send ciphertext/plaintext here).
 * Returns `null` for text secrets with no metadata/hint so the client can sniff JSON vs plain text.
 */
export function resolveMimeServer(input: {
	name: string
	kind: "text" | "binary"
	metadataMime?: string
	typeHint?: string
}): string | null {
	if (input.metadataMime?.trim()) {
		return normalizeMimeServer(input.metadataMime.trim())
	}
	if (input.typeHint?.trim()) {
		return normalizeMimeServer(input.typeHint.trim())
	}
	if (input.kind === "binary") {
		const looked = mime.lookup(input.name)
		if (looked) return normalizeMimeServer(looked)
		return OCTET
	}
	return null
}

export function downloadFilenameServer(name: string, mimeType: string): string {
	const clean = name.trim() || "download"
	if (clean.includes(".") && !clean.endsWith(".")) {
		const last = clean.lastIndexOf(".")
		if (last > 0 && last < clean.length - 1) return clean
	}
	const ext = mime.extension(mimeType.split(";")[0].trim().toLowerCase())
	return ext ? `${clean}.${ext}` : clean
}
