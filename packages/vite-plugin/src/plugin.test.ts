import { beforeEach, describe, expect, it, vi } from "vitest"
import * as loader from "./loader"
import { zenvPlugin } from "./plugin"

vi.mock("./loader", async (importOriginal) => {
	const mod = await importOriginal<typeof import("./loader")>()
	return {
		...mod,
		fetchSecrets: vi.fn(),
		assertCredentials: vi.fn(),
	}
})

describe("zenvPlugin", () => {
	beforeEach(() => {
		vi.resetAllMocks()
	})

	it("returns a Vite plugin with the correct name and enforce flag", () => {
		const plugin = zenvPlugin({ token: "t", projectKey: "pk", projectId: "id", schema: {} })
		expect(plugin.name).toBe("vite-plugin-zenv")
		expect(plugin.enforce).toBe("pre")
	})

	it("config hook fetches secrets and returns them as Vite defines", async () => {
		vi.mocked(loader.fetchSecrets).mockResolvedValue({
			API_KEY: "secret-123",
		})

		const plugin = zenvPlugin({ token: "t", projectKey: "pk", projectId: "id", schema: {} })

		// @ts-expect-error
		const configRes = await plugin.config!({}, { command: "serve" })

		expect(loader.fetchSecrets).toHaveBeenCalled()

		// By default prefix is import.meta.env
		expect(configRes).toEqual({
			define: {
				"import.meta.env.ZENV_API_KEY": JSON.stringify("secret-123"),
			},
		})
	})

	it("aborts build on failure but gracefully degrades on dev server", async () => {
		vi.mocked(loader.fetchSecrets).mockRejectedValue(new Error("Network error"))

		const plugin = zenvPlugin({ token: "t", projectKey: "pk", projectId: "id", schema: {} })

		// In build mode, it should throw
		// @ts-expect-error
		await expect(plugin.config!({}, { command: "build" })).rejects.toThrow(/Failed to load secrets/)

		// In serve mode, it should swallow and return empty define map
		// @ts-expect-error
		const configRes = await plugin.config!({}, { command: "serve" })
		expect(configRes).toEqual({ define: {} })
	})
})
