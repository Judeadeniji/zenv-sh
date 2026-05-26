import { describe, expect, it } from "vitest"
import {
	downloadFilenameServer,
	normalizeMimeServer,
	resolveMimeServer,
} from "#/server/secret-mime-lookup"

describe("secret-mime-lookup (server)", () => {
	it("normalizeMimeServer lowercases", () => {
		expect(normalizeMimeServer("application/JSON")).toBe("application/json")
	})

	it("resolveMimeServer prefers metadata", () => {
		expect(
			resolveMimeServer({
				name: "foo.txt",
				kind: "binary",
				metadataMime: "application/pdf",
			}),
		).toBe("application/pdf")
	})

	it("resolveMimeServer looks up extension for binary", () => {
		expect(
			resolveMimeServer({
				name: "report.pdf",
				kind: "binary",
			}),
		).toBe("application/pdf")
	})

	it("resolveMimeServer returns null for text without hints", () => {
		expect(
			resolveMimeServer({
				name: "env",
				kind: "text",
			}),
		).toBeNull()
	})

	it("resolveMimeServer uses typeHint", () => {
		expect(
			resolveMimeServer({
				name: "blob",
				kind: "binary",
				typeHint: "image/png",
			}),
		).toBe("image/png")
	})

	it("downloadFilenameServer adds extension", () => {
		expect(downloadFilenameServer("out", "application/pdf")).toMatch(/\.pdf$/)
		expect(downloadFilenameServer("file.pdf", "application/pdf")).toBe("file.pdf")
	})
})
