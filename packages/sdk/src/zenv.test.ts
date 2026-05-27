import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { zenv } from "./index"

describe("ZEnv", () => {
	const originalWindow = globalThis.window

	beforeEach(() => {
		// Ensure we are in a "Node" environment
		// @ts-expect-error
		delete globalThis.window
	})

	afterEach(() => {
		globalThis.window = originalWindow
		vi.restoreAllMocks()
	})

	it("throws if instantiated in a browser", () => {
		// @ts-expect-error
		globalThis.window = {}
		expect(() => zenv({ token: "t", projectKey: "pk", projectId: "id" })).toThrowError(/browser/i)
	})

	it("throws if missing credentials", () => {
		expect(() => zenv({ token: "", projectKey: "pk", projectId: "id" })).toThrowError(/ZENV_TOKEN/)
		expect(() => zenv({ token: "t", projectKey: "", projectId: "id" })).toThrowError(
			/ZENV_PROJECT_KEY/,
		)
		expect(() => zenv({ token: "t", projectKey: "pk", projectId: "" })).toThrowError(/projectId/)
	})

	it("initializes without errors when credentials are provided", () => {
		const vault = zenv({ token: "t", projectKey: "pk", projectId: "id" })
		expect(vault).toBeDefined()
	})
})
