import { describe, it, expect } from "vitest"
import { bytesToHex, bytesToBase64, base64ToBytes } from "./encoding"

describe("encoding", () => {
	it("bytesToHex correctly converts bytes to a hex string", () => {
		const bytes = new Uint8Array([0xde, 0xad, 0xbe, 0xef, 0x00, 0xff])
		expect(bytesToHex(bytes)).toBe("deadbeef00ff")
	})

	it("bytesToBase64 correctly encodes bytes", () => {
		const bytes = new Uint8Array([104, 101, 108, 108, 111]) // "hello"
		expect(bytesToBase64(bytes)).toBe("aGVsbG8=")
	})

	it("base64ToBytes correctly decodes base64 strings", () => {
		const b64 = "aGVsbG8="
		const bytes = base64ToBytes(b64)
		expect(Array.from(bytes)).toEqual([104, 101, 108, 108, 111])
	})

	it("roundtrips base64 perfectly", () => {
		const bytes = new Uint8Array([1, 2, 3, 255, 0, 128])
		const encoded = bytesToBase64(bytes)
		const decoded = base64ToBytes(encoded)
		expect(decoded).toEqual(bytes)
	})
})
