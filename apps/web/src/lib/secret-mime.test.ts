import { describe, expect, it } from "vitest"
import {
	downloadFilename,
	inferTextMime,
	isPreviewableMime,
	mimeToFileIconType,
	resolveSecretMime,
} from "#/lib/secret-mime"
import type { DecryptedSecretRow } from "#/lib/queries/secrets"

describe("secret-mime (client)", () => {
	it("inferTextMime detects JSON", () => {
		expect(inferTextMime(`{"a":1}`)).toBe("application/json")
		expect(inferTextMime("hello")).toBe("text/plain")
	})

	it("resolveSecretMime uses server-resolved MIME when present", () => {
		const row = {
			name: "x",
			value: "not-json",
			kind: "text" as const,
			resolvedMime: "application/json",
		}
		expect(resolveSecretMime(row)).toBe("application/json")
	})

	it("resolveSecretMime sniffs text when no resolvedMime", () => {
		const row: Pick<DecryptedSecretRow, "name" | "value" | "kind"> = {
			name: "n",
			value: "{}",
			kind: "text",
		}
		expect(resolveSecretMime(row)).toBe("application/json")
	})

	it("mimeToFileIconType maps common types", () => {
		expect(mimeToFileIconType("application/pdf")).toBe("pdf")
		expect(mimeToFileIconType("image/png")).toBe("png")
		expect(mimeToFileIconType("unknown/thing")).toBe("document")
	})

	it("isPreviewableMime rejects html", () => {
		expect(isPreviewableMime("text/html")).toBe(false)
		expect(isPreviewableMime("text/plain")).toBe(true)
	})

	it("downloadFilename prefers server-suggested name", () => {
		expect(downloadFilename("secret", "application/pdf", "secret.pdf")).toBe("secret.pdf")
		expect(downloadFilename("file.pdf", "application/pdf", null)).toBe("file.pdf")
	})
})
