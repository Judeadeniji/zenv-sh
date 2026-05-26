import { createServerFn } from "@tanstack/react-start"
import { z } from "zod"
import { downloadFilenameServer, resolveMimeServer } from "#/server/secret-mime-lookup"

const batchItemSchema = z.object({
	name: z.string(),
	kind: z.enum(["text", "binary"]),
	metadataMime: z.string().optional(),
	typeHint: z.string().optional(),
})

/** Batch-resolve MIME hints / path lookups for secrets list (no secret values). */
export const resolveSecretMimeBatchServerFn = createServerFn({ method: "POST" })
	.inputValidator((d) => z.object({ items: z.array(batchItemSchema) }).parse(d))
	.handler(async ({ data }) => {
		return data.items.map((item) => {
			const resolved = resolveMimeServer(item)
			if (resolved === null) {
				return { mime: null as string | null, suggestedDownloadFilename: null as string | null }
			}
			return {
				mime: resolved,
				suggestedDownloadFilename: downloadFilenameServer(item.name, resolved),
			}
		})
	})

export const inferFileMimeServerFn = createServerFn({ method: "POST" })
	.inputValidator((d) =>
		z
			.object({
				filename: z.string(),
				typeHint: z.string().optional(),
			})
			.parse(d),
	)
	.handler(async ({ data }) => {
		const resolved = resolveMimeServer({
			name: data.filename,
			kind: "binary",
			typeHint: data.typeHint,
		})
		return resolved ?? "application/octet-stream"
	})

export const inferBinarySecretMimeServerFn = createServerFn({ method: "POST" })
	.inputValidator((d) =>
		z
			.object({
				name: z.string(),
				metadataMime: z.string().optional(),
			})
			.parse(d),
	)
	.handler(async ({ data }) => {
		const resolved = resolveMimeServer({
			name: data.name,
			kind: "binary",
			metadataMime: data.metadataMime,
		})
		return resolved ?? "application/octet-stream"
	})
