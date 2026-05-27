import { defineConfig } from "vitest/config"

export default defineConfig({
	test: {
		// Argon2id hashing tests can take longer than the default 5s in slower environments/CI
		testTimeout: 30000,
	},
})
